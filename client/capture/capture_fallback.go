//go:build !cgo
// +build !cgo

package capture

import (
	"bytes"
	"image"
	"image/png"
	"sync"

	"github.com/kbinani/screenshot"
)

// FallbackCapturer provides screenshot-based capture when CGO is unavailable.
// This is less efficient but works as a fallback.
type FallbackCapturer struct {
	width     int
	height    int
	started   bool
	mu        sync.Mutex
	framePool *FramePool

	prevFrame     []byte
	keyFrameCount int
}

func NewFallbackCapturer() *FallbackCapturer {
	return &FallbackCapturer{
		framePool: NewFramePool(),
	}
}

func (c *FallbackCapturer) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return ErrAlreadyStarted
	}

	bounds := screenshot.GetDisplayBounds(0)
	c.width = bounds.Dx()
	c.height = bounds.Dy()
	c.started = true

	c.prevFrame = make([]byte, c.width*c.height*4)

	return nil
}

func (c *FallbackCapturer) ReadFrame() (*FrameWithDirty, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return nil, ErrNotStarted
	}

	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, ErrCaptureFailed
	}

	if bounds.Dx() != c.width || bounds.Dy() != c.height {
		c.width = bounds.Dx()
		c.height = bounds.Dy()
		c.prevFrame = make([]byte, c.width*c.height*4)
	}

	frame := c.framePool.Get(c.width, c.height)

	copy(frame.Pix, img.Pix)

	c.keyFrameCount++
	isKeyFrame := c.keyFrameCount >= 40
	if isKeyFrame {
		c.keyFrameCount = 0
	}

	result := &FrameWithDirty{
		Frame:      frame,
		IsKeyFrame: isKeyFrame,
	}

	if !isKeyFrame {
		dirtyRects := c.detectDirtyRects(frame.Pix)
		result.DirtyRects = dirtyRects
		if len(dirtyRects) == 0 {
			c.framePool.Put(frame)
			return nil, nil
		}
	}

	copy(c.prevFrame, frame.Pix)

	return result, nil
}

func (c *FallbackCapturer) detectDirtyRects(current []byte) []DirtyRect {
	var rects []DirtyRect
	blockSize := 64
	stride := c.width * 4

	blocksX := (c.width + blockSize - 1) / blockSize
	blocksY := (c.height + blockSize - 1) / blockSize

	for by := 0; by < blocksY; by++ {
		for bx := 0; bx < blocksX; bx++ {
			rx := bx * blockSize
			ry := by * blockSize
			rw := blockSize
			rh := blockSize
			if rx+rw > c.width {
				rw = c.width - rx
			}
			if ry+rh > c.height {
				rh = c.height - ry
			}

			changed := false
			for cy := ry; cy < ry+rh && !changed; cy += 8 {
				for cx := rx; cx < rx+rw && !changed; cx += 8 {
					idx := cy*stride + cx*4
					if idx+4 <= len(current) && idx+4 <= len(c.prevFrame) {
						if current[idx] != c.prevFrame[idx] ||
							current[idx+1] != c.prevFrame[idx+1] ||
							current[idx+2] != c.prevFrame[idx+2] {
							changed = true
						}
					}
				}
			}

			if changed {
				rects = append(rects, DirtyRect{X: rx, Y: ry, W: rw, H: rh})
			}
		}
	}

	return rects
}

func (c *FallbackCapturer) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.started = false
}

func (c *FallbackCapturer) SupportsDirtyRects() bool {
	return false
}

func NewCapturer() (Capturer, error) {
	return NewFallbackCapturer(), nil
}

func imageToRGBA(img image.Image) []byte {
	var buf bytes.Buffer
	png.Encode(&buf, img)

	bounds := img.Bounds()
	rgba := make([]byte, bounds.Dx()*bounds.Dy()*4)
	stride := bounds.Dx() * 4

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			idx := (y-bounds.Min.Y)*stride + (x-bounds.Min.X)*4
			rgba[idx] = uint8(r >> 8)
			rgba[idx+1] = uint8(g >> 8)
			rgba[idx+2] = uint8(b >> 8)
			rgba[idx+3] = uint8(a >> 8)
		}
	}

	return rgba
}
