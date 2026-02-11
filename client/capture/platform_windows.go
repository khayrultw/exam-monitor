//go:build windows && cgo
// +build windows,cgo

package capture

func NewPlatformCapturer() Capturer {
	return NewDXGICapturer()
}
