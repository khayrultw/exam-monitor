# Exam Monitor

A real-time exam monitoring system with **compositor-based screen capture**. The client captures the student's screen using platform-native APIs and streams it to the server for live monitoring.

## Architecture

### Screen Capture System

The client uses efficient, compositor-based screen capture via CGO for each platform:

| Platform | API | Features |
|----------|-----|----------|
| **Linux X11** | XShm + XDamage | Shared memory capture, dirty rectangle detection |
| **Linux Wayland** | PipeWire | Portal-based capture with damage regions |
| **Windows** | DXGI Desktop Duplication | GPU-accelerated, native dirty rect support |
| **macOS** | CGDisplayStream | Low-latency compositor capture |

### Frame Encoding

- **MJPEG encoding** with quality 45 (configurable)
- **Dirty rectangle optimization**: Only changed regions are encoded after keyframes
- **Keyframe interval**: Every 5 seconds (30 frames at 6 FPS)
- **Max width scaling**: Frames scaled to 720px for bandwidth efficiency
- **Wire protocol**: 8-byte header (`HE` + type:2 + length:4)

### Wire Format

Frames are transmitted with a type byte prefix:
- `0x01` - Keyframe: Full JPEG image
- `0x02` - Dirty rectangles: Header + multiple JPEG tiles

```
Keyframe:     [0x01][JPEG data...]
Dirty frame:  [0x02][count:2][x:2][y:2][w:2][h:2][len:4][JPEG]...
```

## Building

### Prerequisites

#### Linux (X11)
```bash
sudo apt install libvulkan-dev libxkbcommon-x11-dev libx11-xcb-dev \
    libx11-dev libxext-dev libxdamage-dev
```

#### Linux (Wayland)
```bash
sudo apt install libpipewire-0.3-dev
# Build with: go build -tags wayland
```

#### macOS
```bash
brew install go
```

#### Windows
Requires MinGW for cross-compilation from Linux.

### Build Client

```bash
cd client
go build -o client .
```

### Build Server

```bash
cd server
go build -o server .
```

### Cross-compile for Windows (from Linux)

```bash
cd client
x86_64-w64-mingw32-windres app.rc -O coff -o app-res.o
GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc \
    go build -ldflags "-H=windowsgui -extldflags=-Wl,app-res.o" -o examgaurd.exe .
```

## Running

### Server
```bash
cd server && ./server
```

### Client
```bash
cd client && ./client
```

## Downloads

- **Windows Client**: [examgaurd-student-v1.0.0](https://github.com/khayrultw/exam-monitor/releases/download/v1.0.0-rc/examgaurd_student.exe)
- **Windows Server**: [examgaurd-teacher-v1.0.0](https://github.com/khayrultw/exam-monitor/releases/download/v1.0.0-rc/examgaurd_teacher)

## Performance

- **Capture rate**: 6 FPS (configurable)
- **Memory efficient**: Uses sync.Pool for buffer reuse
- **Bandwidth optimized**: Dirty rectangles reduce data by 60-80%
- **Low latency**: Direct compositor access, no intermediate copies

