//go:build linux && wayland && cgo
// +build linux,wayland,cgo

package capture

func NewPlatformCapturer() Capturer {
	return NewWaylandCapturer()
}
