//go:build darwin
// +build darwin

package capture

func NewPlatformCapturer() Capturer {
	return NewCGCapturer()
}
