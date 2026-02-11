//go:build darwin && cgo
// +build darwin,cgo

package capture

func NewPlatformCapturer() Capturer {
	return NewCGCapturer()
}
