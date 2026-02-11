//go:build linux && !wayland && cgo
// +build linux,!wayland,cgo

package capture

import "os"

// NewPlatformCapturer creates the platform-specific capturer for Linux.
// On Wayland (detected via XDG_SESSION_TYPE), uses the portal capturer.
// On X11, uses the X11 capturer.
func NewPlatformCapturer() Capturer {
	if os.Getenv("XDG_SESSION_TYPE") == "wayland" {
		return NewPortalCapturer()
	}
	return NewX11Capturer()
}
