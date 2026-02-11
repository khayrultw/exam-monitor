// Package encoder provides MJPEG encoding with dirty rectangle support.
package encoder

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/jpeg"
	"sync"

	"github.com/exam-gaurd/client/capture"
)

// EncodedFrame represents an encoded frame ready for transmission.
type EncodedFrame struct {
	// Data contains the encoded frame data.
	// For keyframes: just JPEG data
	// For dirty rect frames: header + multiple JPEG tiles
	Data []byte

	// IsKeyFrame indicates if this is a full frame.
	IsKeyFrame bool
}

// FrameType constants for protocol
const (
	FrameTypeKey   byte = 0x01 // Full JPEG frame
	FrameTypeDirty byte = 0x02 // Dirty rectangles
)

// Encoder handles MJPEG encoding with dirty rectangle optimization.
type Encoder struct {
	quality     int
	maxWidth    int
	bufferPool  *sync.Pool
	jpegBufPool *sync.Pool

	// Statistics
	keyFramesSent   int64
	dirtyFramesSent int64
}

// EncoderConfig holds encoder configuration.
type EncoderConfig struct {
	// Quality is the JPEG quality (1-100). Default: 45
	Quality int

	// MaxWidth is the maximum output width. 0 means no scaling.
	MaxWidth int
}

// DefaultConfig returns the default encoder configuration.
func DefaultConfig() EncoderConfig {
	return EncoderConfig{
		Quality:  45,
		MaxWidth: 720,
	}
}

// NewEncoder creates a new frame encoder.
func NewEncoder(config EncoderConfig) *Encoder {
	if config.Quality <= 0 || config.Quality > 100 {
		config.Quality = 45
	}

	return &Encoder{
		quality:  config.Quality,
		maxWidth: config.MaxWidth,
		bufferPool: &sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, 128*1024))
			},
		},
		jpegBufPool: &sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, 32*1024))
			},
		},
	}
}

// Encode encodes a frame with optional dirty rectangles.
// Returns nil if the frame should be dropped (no changes).
func (e *Encoder) Encode(frame *capture.FrameWithDirty) (*EncodedFrame, error) {
	if frame == nil || frame.Frame == nil {
		return nil, nil
	}

	buf := e.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer e.bufferPool.Put(buf)

	if frame.IsKeyFrame || len(frame.DirtyRects) == 0 {
		// Encode full frame as keyframe
		return e.encodeKeyFrame(frame.Frame, buf)
	}

	// Encode dirty rectangles
	return e.encodeDirtyRects(frame.Frame, frame.DirtyRects, buf)
}

// encodeKeyFrame encodes a full frame as JPEG.
func (e *Encoder) encodeKeyFrame(frame *capture.Frame, buf *bytes.Buffer) (*EncodedFrame, error) {
	// Create image from frame
	img := e.frameToImage(frame)

	// Scale if needed
	if e.maxWidth > 0 && img.Bounds().Dx() > e.maxWidth {
		img = e.scaleImage(img, e.maxWidth)
	}

	// Write frame type header
	buf.WriteByte(FrameTypeKey)

	// Encode as JPEG
	opts := jpeg.Options{Quality: e.quality}
	if err := jpeg.Encode(buf, img, &opts); err != nil {
		return nil, err
	}

	// Copy result
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())

	e.keyFramesSent++

	return &EncodedFrame{
		Data:       result,
		IsKeyFrame: true,
	}, nil
}

// encodeDirtyRects encodes only changed regions.
// Format: [type:1][count:2][rect1_header:8][rect1_data]...[rectN_header:8][rectN_data]
// Rect header: [x:2][y:2][w:2][h:2]
func (e *Encoder) encodeDirtyRects(frame *capture.Frame, rects []capture.DirtyRect, buf *bytes.Buffer) (*EncodedFrame, error) {
	// Calculate scale factor
	scaleX := 1.0
	scaleY := 1.0
	if e.maxWidth > 0 && frame.W > e.maxWidth {
		scale := float64(e.maxWidth) / float64(frame.W)
		scaleX = scale
		scaleY = scale
	}

	// Write header
	buf.WriteByte(FrameTypeDirty)

	// Write rect count (will update later if some are skipped)
	countPos := buf.Len()
	binary.Write(buf, binary.BigEndian, uint16(0))

	jpegBuf := e.jpegBufPool.Get().(*bytes.Buffer)
	defer e.jpegBufPool.Put(jpegBuf)

	actualCount := uint16(0)

	for _, rect := range rects {
		// Extract rect pixels from frame
		rectImg := e.extractRect(frame, rect)
		if rectImg == nil {
			continue
		}

		// Scale rect coordinates and size
		scaledX := int(float64(rect.X) * scaleX)
		scaledY := int(float64(rect.Y) * scaleY)
		scaledW := int(float64(rect.W) * scaleX)
		scaledH := int(float64(rect.H) * scaleY)

		// Scale rect image if needed
		if scaleX != 1.0 {
			rectImg = e.scaleImage(rectImg, scaledW)
		}

		// Encode rect as JPEG
		jpegBuf.Reset()
		opts := jpeg.Options{Quality: e.quality + 5} // Slightly higher quality for small regions
		if err := jpeg.Encode(jpegBuf, rectImg, &opts); err != nil {
			continue
		}

		// Write rect header: x, y, w, h (scaled coordinates)
		binary.Write(buf, binary.BigEndian, uint16(scaledX))
		binary.Write(buf, binary.BigEndian, uint16(scaledY))
		binary.Write(buf, binary.BigEndian, uint16(scaledW))
		binary.Write(buf, binary.BigEndian, uint16(scaledH))

		// Write JPEG data length and data
		binary.Write(buf, binary.BigEndian, uint32(jpegBuf.Len()))
		buf.Write(jpegBuf.Bytes())

		actualCount++
	}

	// Update rect count
	data := buf.Bytes()
	binary.BigEndian.PutUint16(data[countPos:], actualCount)

	// If no rects were encoded, return nil
	if actualCount == 0 {
		return nil, nil
	}

	// Copy result
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())

	e.dirtyFramesSent++

	return &EncodedFrame{
		Data:       result,
		IsKeyFrame: false,
	}, nil
}

// frameToImage converts a capture.Frame to image.Image.
func (e *Encoder) frameToImage(frame *capture.Frame) image.Image {
	return &image.RGBA{
		Pix:    frame.Pix,
		Stride: frame.Stride,
		Rect:   image.Rect(0, 0, frame.W, frame.H),
	}
}

// extractRect extracts a rectangular region from a frame.
func (e *Encoder) extractRect(frame *capture.Frame, rect capture.DirtyRect) image.Image {
	// Bounds check
	if rect.X < 0 || rect.Y < 0 || rect.X+rect.W > frame.W || rect.Y+rect.H > frame.H {
		return nil
	}

	// Create sub-image
	// For RGBA, we need to calculate the proper offset
	startIdx := rect.Y*frame.Stride + rect.X*4

	// Create new RGBA with the rect data
	rectPix := make([]byte, rect.W*4*rect.H)
	rectStride := rect.W * 4

	for y := 0; y < rect.H; y++ {
		srcStart := startIdx + y*frame.Stride
		srcEnd := srcStart + rect.W*4
		dstStart := y * rectStride

		if srcEnd <= len(frame.Pix) {
			copy(rectPix[dstStart:dstStart+rect.W*4], frame.Pix[srcStart:srcEnd])
		}
	}

	return &image.RGBA{
		Pix:    rectPix,
		Stride: rectStride,
		Rect:   image.Rect(0, 0, rect.W, rect.H),
	}
}

// scaleImage scales an image to the specified width, maintaining aspect ratio.
// Uses simple nearest-neighbor scaling for speed.
func (e *Encoder) scaleImage(src image.Image, targetWidth int) image.Image {
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	if srcW <= targetWidth {
		return src
	}

	scale := float64(targetWidth) / float64(srcW)
	targetHeight := int(float64(srcH) * scale)

	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))

	// Nearest-neighbor scaling
	for y := 0; y < targetHeight; y++ {
		srcY := int(float64(y) / scale)
		if srcY >= srcH {
			srcY = srcH - 1
		}
		for x := 0; x < targetWidth; x++ {
			srcX := int(float64(x) / scale)
			if srcX >= srcW {
				srcX = srcW - 1
			}

			c := src.At(bounds.Min.X+srcX, bounds.Min.Y+srcY)
			dst.Set(x, y, c)
		}
	}

	return dst
}

// Stats returns encoder statistics.
func (e *Encoder) Stats() (keyFrames, dirtyFrames int64) {
	return e.keyFramesSent, e.dirtyFramesSent
}
