//go:build linux && wayland
// +build linux,wayland

package capture

func NewPlatformCapturer() Capturer {
	return NewWaylandCapturer()
}
