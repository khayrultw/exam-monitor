//go:build !cgo
// +build !cgo

package capture

func NewPlatformCapturer() Capturer {
	return NewFallbackCapturer()
}
