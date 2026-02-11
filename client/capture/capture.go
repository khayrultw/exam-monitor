// Package capture provides a cross-platform screen capture interface
// with support for dirty rectangle detection and efficient memory usage.
package capture

import (
	"errors"
	"sync"
)

// Frame represents a captured screen frame.
// Pix contains RGBA pixel data, organized as 4 bytes per pixel.
type Frame struct {
	Pix    []byte // RGBA pixel data
	W, H   int    // Width and height in pixels
	Stride int    // Bytes per row (typically W * 4 for RGBA)
}

// DirtyRect represents a changed region of the screen.
type DirtyRect struct {
	X, Y, W, H int
}

// FrameWithDirty contains a frame and optional dirty rectangles.
// If DirtyRects is empty, the entire frame should be considered changed.
type FrameWithDirty struct {
	Frame      *Frame
	DirtyRects []DirtyRect
	IsKeyFrame bool // True if this is a full frame (no dirty rect optimization)
}

// Capturer is the interface for platform-specific screen capture.
type Capturer interface {
	// Start initializes the capture system.
	// Must be called before ReadFrame.
	Start() error

	// ReadFrame captures the current screen state.
	// Returns a frame with optional dirty rectangles.
	// The Frame's Pix slice may be reused between calls for efficiency.
	ReadFrame() (*FrameWithDirty, error)

	// Stop releases capture resources.
	// After Stop, the capturer cannot be reused.
	Stop()

	// SupportssDirtyRects returns true if the capturer supports dirty rectangle detection.
	SupportsDirtyRects() bool
}

// Common errors
var (
	ErrNotStarted       = errors.New("capture: not started")
	ErrAlreadyStarted   = errors.New("capture: already started")
	ErrCaptureFailed    = errors.New("capture: failed to capture frame")
	ErrNotSupported     = errors.New("capture: not supported on this platform")
	ErrNoDisplay        = errors.New("capture: no display available")
	ErrPermissionDenied = errors.New("capture: permission denied")
)

// FramePool provides pooled Frame allocations to reduce GC pressure.
type FramePool struct {
	pool sync.Pool
}

// NewFramePool creates a new frame pool.
func NewFramePool() *FramePool {
	return &FramePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &Frame{}
			},
		},
	}
}

func (p *FramePool) Get(w, h int) *Frame {
	f := p.pool.Get().(*Frame)
	stride := w * 4
	size := stride * h

	if cap(f.Pix) < size {
		f.Pix = make([]byte, size)
	} else {
		f.Pix = f.Pix[:size]
	}

	f.W = w
	f.H = h
	f.Stride = stride
	return f
}

func (p *FramePool) Put(f *Frame) {
	if f != nil {
		p.pool.Put(f)
	}
}

type BytePool struct {
	pool sync.Pool
	size int
}

func NewBytePool(size int) *BytePool {
	return &BytePool{
		size: size,
		pool: sync.Pool{
			New: func() interface{} {
				b := make([]byte, 0, size)
				return &b
			},
		},
	}
}

func (p *BytePool) Get() *[]byte {
	return p.pool.Get().(*[]byte)
}

func (p *BytePool) Put(b *[]byte) {
	if b != nil {
		*b = (*b)[:0]
		p.pool.Put(b)
	}
}
