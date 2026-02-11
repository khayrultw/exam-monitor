//go:build windows
// +build windows

package capture

func NewPlatformCapturer() Capturer {
	return NewDXGICapturer()
}
