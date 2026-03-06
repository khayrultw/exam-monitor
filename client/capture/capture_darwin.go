//go:build darwin && cgo
// +build darwin,cgo

package capture

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
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
	prevFrame  []byte
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

	// Check screen recording permission before initializing
	if C.sck_check_screen_recording_permission() == 0 {
		return errors.New("Screen Recording permission required. Open System Settings > Privacy & Security > Screen Recording and enable this app, then retry.")
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

	var width, height C.int
	result := C.sck_capture_get_frame(
		c.cap,
		(*C.uint8_t)(unsafe.Pointer(&c.rgbaBuffer[0])),
		C.int(len(c.rgbaBuffer)),
		&width, &height,
	)

	// Buffer too small (e.g., display resolution changed), reallocate and retry
	if result == -2 {
		c.width = int(width)
		c.height = int(height)
		c.rgbaBuffer = make([]byte, c.width*c.height*4)
		c.prevFrame = nil

		result = C.sck_capture_get_frame(
			c.cap,
			(*C.uint8_t)(unsafe.Pointer(&c.rgbaBuffer[0])),
			C.int(len(c.rgbaBuffer)),
			&width, &height,
		)
	}

	if result == 0 {
		return nil, nil // No new frame
	}
	if result < 0 {
		return nil, ErrCaptureFailed
	}

	c.width = int(width)
	c.height = int(height)
	pixelBytes := c.width * c.height * 4

	frame := c.framePool.Get(c.width, c.height)
	copy(frame.Pix, c.rgbaBuffer[:pixelBytes])

	// First frame is always a keyframe; periodic keyframes are handled by client.go
	isKeyFrame := c.prevFrame == nil

	var dirtyRects []DirtyRect
	if !isKeyFrame {
		dirtyRects = c.detectDirtyRects(frame.Pix)
		if len(dirtyRects) == 0 {
			c.framePool.Put(frame)
			return nil, nil // No changes
		}
	}

	// Save current frame for next dirty rect comparison
	if c.prevFrame == nil || len(c.prevFrame) != pixelBytes {
		c.prevFrame = make([]byte, pixelBytes)
	}
	copy(c.prevFrame, frame.Pix)

	return &FrameWithDirty{
		Frame:      frame,
		DirtyRects: dirtyRects,
		IsKeyFrame: isKeyFrame,
	}, nil
}

func (c *SCKCapturer) detectDirtyRects(current []byte) []DirtyRect {
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
	return true
}
