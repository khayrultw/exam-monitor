//go:build ignore
// +build ignore

// Simple test program to verify screen capture is working.
// Run with: go run capturecheck.go
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/exam-gaurd/client/capture"
)

func main() {
	fmt.Println("Testing screen capture...")
	fmt.Printf("XDG_SESSION_TYPE: %s\n", os.Getenv("XDG_SESSION_TYPE"))

	// Create platform capturer
	cap := capture.NewPlatformCapturer()
	fmt.Printf("Capturer type: %T\n", cap)

	// Start capture
	if err := cap.Start(); err != nil {
		fmt.Printf("Failed to start capture: %v\n", err)
		os.Exit(1)
	}
	defer cap.Stop()

	fmt.Println("Capture started, reading frame...")

	// Read a frame
	frameData, err := cap.ReadFrame()
	if err != nil {
		fmt.Printf("Failed to read frame: %v\n", err)
		os.Exit(1)
	}

	if frameData == nil || frameData.Frame == nil {
		fmt.Println("No frame data returned!")
		os.Exit(1)
	}

	frame := frameData.Frame
	fmt.Printf("Frame: %dx%d, stride=%d, pix_len=%d\n",
		frame.W, frame.H, frame.Stride, len(frame.Pix))

	// Check if frame is all black
	nonZeroCount := 0
	for i := 0; i < len(frame.Pix) && i < 10000; i++ {
		if frame.Pix[i] != 0 {
			nonZeroCount++
		}
	}
	fmt.Printf("Non-zero bytes in first 10000: %d\n", nonZeroCount)

	// Create image and save to PNG
	img := &image.RGBA{
		Pix:    frame.Pix,
		Stride: frame.Stride,
		Rect:   image.Rect(0, 0, frame.W, frame.H),
	}

	outFile, err := os.Create("capture_test.png")
	if err != nil {
		fmt.Printf("Failed to create output file: %v\n", err)
		os.Exit(1)
	}
	defer outFile.Close()

	if err := png.Encode(outFile, img); err != nil {
		fmt.Printf("Failed to encode PNG: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Saved to capture_test.png - check if image is correct!")
}
