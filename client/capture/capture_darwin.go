//go:build darwin && cgo
// +build darwin,cgo

package capture

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework ScreenCaptureKit -framework CoreGraphics -framework CoreMedia -framework CoreVideo -framework Foundation

#include "screencapturekit_darwin.h"
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"sync"
	"time"
	"unsafe"
)

type SCKCapturer struct {
	cap       *C.SCKCapture
	width     int
	height    int
	started   bool
	mu        sync.Mutex
	framePool *FramePool

	rgbaBuffer []byte

	keyFrameCounter int
}

func NewSCKCapturer() *SCKCapturer {
	return &SCKCapturer{
		framePool: NewFramePool(),
	}
}

func (c *SCKCapturer) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return ErrAlreadyStarted
	}

	var width, height C.int
	var errMsg *C.char

	c.cap = C.sck_capture_init(&width, &height, &errMsg)
	if c.cap == nil {
		if errMsg != nil {
			msg := C.GoString(errMsg)
			C.free(unsafe.Pointer(errMsg))
			return errors.New("ScreenCaptureKit init failed: " + msg)
		}
		return ErrNoDisplay
	}

	c.width = int(width)
	c.height = int(height)
	c.rgbaBuffer = make([]byte, c.width*c.height*4)

	// Start capturing
	if C.sck_capture_start(c.cap) != 1 {
		C.sck_capture_destroy(c.cap)
		c.cap = nil
		return errors.New("failed to start capture")
	}

	c.started = true

	// Wait for first frame
	time.Sleep(500 * time.Millisecond)

	return nil
}

func (c *SCKCapturer) ReadFrame() (*FrameWithDirty, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return nil, ErrNotStarted
	}

	// Check if dimensions changed
	var width, height C.int
	C.sck_capture_get_size(c.cap, &width, &height)
	if int(width) != c.width || int(height) != c.height {
		c.width = int(width)
		c.height = int(height)
		c.rgbaBuffer = make([]byte, c.width*c.height*4)
	}

	// Get frame
	result := C.sck_capture_get_frame(
		c.cap,
		(*C.uint8_t)(unsafe.Pointer(&c.rgbaBuffer[0])),
	)

	if result == 0 {
		return nil, nil // No new frame
	}

	if result < 0 {
		return nil, ErrCaptureFailed
	}

	// Copy to frame
	frame := c.framePool.Get(c.width, c.height)
	copy(frame.Pix, c.rgbaBuffer)

	c.keyFrameCounter++
	forceKeyFrame := c.keyFrameCounter >= 40 // Every ~5 seconds at 8 FPS
	if forceKeyFrame {
		c.keyFrameCounter = 0
	}

	return &FrameWithDirty{
		Frame:      frame,
		IsKeyFrame: forceKeyFrame,
		DirtyRects: nil, // ScreenCaptureKit doesn't provide dirty rects
	}, nil
}

func (c *SCKCapturer) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started && c.cap != nil {
		C.sck_capture_stop(c.cap)
		C.sck_capture_destroy(c.cap)
		c.cap = nil
		c.started = false
	}
}

func (c *SCKCapturer) SupportsDirtyRects() bool {
	return false
}
