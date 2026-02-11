# Screen Capture System

This package provides high-performance, cross-platform screen capture using compositor-based APIs with CGO.

## Overview

The capture system replaces the previous screenshot-based approach with real-time, compositor-level screen capture that provides:

- **Low CPU usage**: Direct access to compositor buffers
- **Low memory allocation**: Buffer pooling and reuse
- **Dirty rectangle detection**: Only encode/transmit changed regions
- **5-8 FPS with periodic keyframes**: Balanced quality and bandwidth

## Platform Support

| Platform | Backend | Dirty Rects | Notes |
|----------|---------|-------------|-------|
| Linux X11 | XDamage + XShm | Native | Best performance on X11 |
| Linux Wayland | PipeWire | Software | Requires portal permission |
| Windows | DXGI Desktop Duplication | Native | Requires Windows 8+ |
| macOS | CGDisplayStream | Software | Requires screen recording permission |
| Fallback | github.com/kbinani/screenshot | Software | No CGO required |

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Capturer   │────▶│   Encoder   │────▶│  Transport  │────▶│   Server    │
│  (CGO)      │     │  (MJPEG)    │     │  (TCP/QUIC) │     │  (Decoder)  │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
      │                    │                    │                   │
      ▼                    ▼                    ▼                   ▼
   Frame +            EncodedFrame        Length-prefixed      Composited
   DirtyRects         (key/dirty)         packets              Image
```

## Interfaces

### Capturer

```go
type Frame struct {
    Pix    []byte // RGBA pixel data
    W, H   int    // Dimensions
    Stride int    // Bytes per row
}

type DirtyRect struct {
    X, Y, W, H int
}

type FrameWithDirty struct {
    Frame      *Frame
    DirtyRects []DirtyRect
    IsKeyFrame bool
}

type Capturer interface {
    Start() error
    ReadFrame() (*FrameWithDirty, error)
    Stop()
    SupportsDirtyRects() bool
}
```

### Encoder

```go
type EncodedFrame struct {
    Data       []byte
    IsKeyFrame bool
}

// Frame types
FrameTypeKey   = 0x01  // Full JPEG
FrameTypeDirty = 0x02  // Multiple JPEG tiles
```

## Wire Protocol

### Keyframe Format
```
[type:1][jpeg_data:N]
```

### Dirty Rect Frame Format
```
[type:1][count:2][rect1_header:8][rect1_len:4][rect1_jpeg:N]...
```

Rect header: `[x:2][y:2][w:2][h:2]`

## Usage

### Client

```go
import "github.com/exam-gaurd/client/capture"

// Create platform-specific capturer
cap := capture.NewX11Capturer() // or NewDXGICapturer(), NewCGCapturer()

if err := cap.Start(); err != nil {
    log.Fatal(err)
}
defer cap.Stop()

// Capture loop
for {
    frame, err := cap.ReadFrame()
    if err != nil {
        break
    }
    if frame == nil {
        continue // No new frame
    }
    
    // Encode and send...
}
```

### Server

The server automatically handles both keyframes and dirty rect frames:

```go
// In handleStudent()
img := s.decodeFrame(id, data)
if img != nil {
    s.studentUtil.UpdateImage(id, img)
}
```

## Build Tags

- `linux,!wayland` - X11 capturer (default on Linux)
- `linux,wayland` - Wayland/PipeWire capturer
- `windows` - DXGI capturer
- `darwin` - CGDisplayStream capturer
- `!cgo` - Fallback screenshot capturer

## Build Requirements

### Linux X11
```bash
sudo apt install libx11-dev libxext-dev libxdamage-dev
```

### Linux Wayland
```bash
sudo apt install libpipewire-0.3-dev
```
Build with: `go build -tags wayland`

### Windows
Requires Windows 8+ for Desktop Duplication API.

### macOS
Requires Screen Recording permission in System Preferences.

## Performance Tuning

- **FPS**: 5-8 recommended for classroom monitoring
- **JPEG Quality**: 45-50 for good balance
- **Keyframe Interval**: Every 30-40 frames (5 seconds at 6-8 FPS)
- **Send Queue**: 2-3 frames, drop if full
- **Block Size**: 64px for dirty rect detection

## Memory Management

- Frame pools via `sync.Pool` reduce GC pressure
- Preallocated RGBA buffers reused between captures
- Double-buffering in macOS for thread safety
- Dirty rect frames reduce bandwidth by ~60-80%
