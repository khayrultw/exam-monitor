//go:build linux
// +build linux

package capture

import (
	"fmt"
	"image"
	_ "image/png" // Register PNG decoder
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

// PortalCapturer uses XDG Desktop Portal for Wayland screen capture.
// This works on all Wayland compositors (KDE, GNOME, etc.)
type PortalCapturer struct {
	conn      *dbus.Conn
	width     int
	height    int
	started   bool
	mu        sync.Mutex
	framePool *FramePool
	tempDir   string

	prevFrame []byte
}

func NewPortalCapturer() *PortalCapturer {
	return &PortalCapturer{
		framePool: NewFramePool(),
	}
}

func (c *PortalCapturer) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return ErrAlreadyStarted
	}

	conn, err := dbus.SessionBus()
	if err != nil {
		return fmt.Errorf("failed to connect to session bus: %w", err)
	}
	c.conn = conn

	c.tempDir, err = os.MkdirTemp("", "capture-*")
	if err != nil {
		c.conn.Close()
		return err
	}

	c.started = true
	return nil
}

func (c *PortalCapturer) ReadFrame() (*FrameWithDirty, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return nil, ErrNotStarted
	}

	// Use spectacle (KDE) or gnome-screenshot with file output
	// as a more reliable method than portal for repeated captures
	img, err := c.captureWithTool()
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	c.width = bounds.Dx()
	c.height = bounds.Dy()

	frame := c.framePool.Get(c.width, c.height)

	c.imageToFrame(img, frame)

	result := &FrameWithDirty{
		Frame:      frame,
		IsKeyFrame: true,
	}

	if c.prevFrame != nil && len(c.prevFrame) == len(frame.Pix) {
		dirtyRects := c.detectDirtyRects(frame)
		if len(dirtyRects) > 0 && len(dirtyRects) < 20 {
			result.DirtyRects = dirtyRects
			result.IsKeyFrame = false
		}
	}

	if c.prevFrame == nil || len(c.prevFrame) != len(frame.Pix) {
		c.prevFrame = make([]byte, len(frame.Pix))
	}
	copy(c.prevFrame, frame.Pix)

	return result, nil
}

func (c *PortalCapturer) captureWithTool() (image.Image, error) {
	tmpFile := filepath.Join(c.tempDir, fmt.Sprintf("cap_%d.png", time.Now().UnixNano()))
	defer os.Remove(tmpFile)

	if err := exec.Command("spectacle", "-b", "-n", "-o", tmpFile).Run(); err == nil {
		return c.loadImage(tmpFile)
	}

	if err := exec.Command("gnome-screenshot", "-f", tmpFile).Run(); err == nil {
		return c.loadImage(tmpFile)
	}

	if err := exec.Command("scrot", tmpFile).Run(); err == nil {
		return c.loadImage(tmpFile)
	}

	return nil, fmt.Errorf("no screenshot tool available (tried spectacle, gnome-screenshot, scrot)")
}

func (c *PortalCapturer) loadImage(path string) (image.Image, error) {
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(path); err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	return img, err
}

func (c *PortalCapturer) imageToFrame(img image.Image, frame *Frame) {
	bounds := img.Bounds()
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			idx := y*frame.Stride + x*4
			frame.Pix[idx+0] = byte(r >> 8)
			frame.Pix[idx+1] = byte(g >> 8)
			frame.Pix[idx+2] = byte(b >> 8)
			frame.Pix[idx+3] = byte(a >> 8)
		}
	}
}

func (c *PortalCapturer) detectDirtyRects(frame *Frame) []DirtyRect {
	var rects []DirtyRect
	blockSize := 64
	blocksX := (frame.W + blockSize - 1) / blockSize
	blocksY := (frame.H + blockSize - 1) / blockSize

	for by := 0; by < blocksY && len(rects) < 32; by++ {
		for bx := 0; bx < blocksX && len(rects) < 32; bx++ {
			rx := bx * blockSize
			ry := by * blockSize
			rw := min(blockSize, frame.W-rx)
			rh := min(blockSize, frame.H-ry)

			changed := false
			for cy := ry; cy < ry+rh && !changed; cy += 8 {
				for cx := rx; cx < rx+rw && !changed; cx += 8 {
					idx := cy*frame.Stride + cx*4
					if idx+4 <= len(frame.Pix) && idx+4 <= len(c.prevFrame) {
						for i := 0; i < 4; i++ {
							if frame.Pix[idx+i] != c.prevFrame[idx+i] {
								changed = true
								break
							}
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

// Stop releases resources.
func (c *PortalCapturer) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		if c.tempDir != "" {
			os.RemoveAll(c.tempDir)
		}
		if c.conn != nil {
			c.conn.Close()
		}
		c.started = false
	}
}

func (c *PortalCapturer) SupportsDirtyRects() bool {
	return true
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
