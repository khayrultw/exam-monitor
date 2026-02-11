//go:build darwin
// +build darwin

package capture

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation -framework Foundation -framework IOSurface

#include <stdlib.h>
#include <string.h>
#include <CoreGraphics/CoreGraphics.h>
#include <IOSurface/IOSurface.h>
#include <dispatch/dispatch.h>

// macOS Display Stream capture state
typedef struct {
    CGDisplayStreamRef stream;
    dispatch_queue_t queue;

    // Frame buffer (double buffered for thread safety)
    unsigned char *frame_buffer[2];
    int current_buffer;
    int frame_width;
    int frame_height;
    int frame_ready;
    int running;

    // Previous frame for dirty rect detection
    unsigned char *prev_frame;

    // Lock for thread-safe access
    pthread_mutex_t mutex;
} CGCapture;

// Frame update callback
void cg_capture_callback(
    CGDisplayStreamFrameStatus status,
    uint64_t displayTime,
    IOSurfaceRef frameSurface,
    CGDisplayStreamUpdateRef updateRef,
    void *userInfo
) {
    CGCapture *cap = (CGCapture*)userInfo;

    if (!cap || !cap->running) return;
    if (status != kCGDisplayStreamFrameStatusFrameComplete) return;
    if (!frameSurface) return;

    // Lock the surface for CPU access
    IOSurfaceLock(frameSurface, kIOSurfaceLockReadOnly, NULL);

    size_t width = IOSurfaceGetWidth(frameSurface);
    size_t height = IOSurfaceGetHeight(frameSurface);
    size_t bytesPerRow = IOSurfaceGetBytesPerRow(frameSurface);
    void *baseAddress = IOSurfaceGetBaseAddress(frameSurface);

    pthread_mutex_lock(&cap->mutex);

    // Resize buffers if needed
    if ((int)width != cap->frame_width || (int)height != cap->frame_height) {
        cap->frame_width = (int)width;
        cap->frame_height = (int)height;

        size_t bufSize = width * height * 4;
        for (int i = 0; i < 2; i++) {
            if (cap->frame_buffer[i]) free(cap->frame_buffer[i]);
            cap->frame_buffer[i] = (unsigned char*)malloc(bufSize);
        }
        if (cap->prev_frame) free(cap->prev_frame);
        cap->prev_frame = (unsigned char*)malloc(bufSize);
    }

    // Copy frame data (convert BGRA to RGBA)
    int nextBuffer = 1 - cap->current_buffer;
    unsigned char *dst = cap->frame_buffer[nextBuffer];
    unsigned char *src = (unsigned char*)baseAddress;
    int dstStride = cap->frame_width * 4;

    for (int y = 0; y < cap->frame_height; y++) {
        for (int x = 0; x < cap->frame_width; x++) {
            int srcIdx = y * bytesPerRow + x * 4;
            int dstIdx = y * dstStride + x * 4;

            // macOS uses BGRA format
            dst[dstIdx + 0] = src[srcIdx + 2]; // R
            dst[dstIdx + 1] = src[srcIdx + 1]; // G
            dst[dstIdx + 2] = src[srcIdx + 0]; // B
            dst[dstIdx + 3] = 255;             // A
        }
    }

    cap->current_buffer = nextBuffer;
    cap->frame_ready = 1;

    pthread_mutex_unlock(&cap->mutex);

    IOSurfaceUnlock(frameSurface, kIOSurfaceLockReadOnly, NULL);
}

// Initialize capture
CGCapture* cg_capture_init(int *out_width, int *out_height) {
    CGCapture *cap = (CGCapture*)calloc(1, sizeof(CGCapture));
    if (!cap) return NULL;

    pthread_mutex_init(&cap->mutex, NULL);

    // Get main display info
    CGDirectDisplayID displayID = CGMainDisplayID();
    size_t width = CGDisplayPixelsWide(displayID);
    size_t height = CGDisplayPixelsHigh(displayID);

    cap->frame_width = (int)width;
    cap->frame_height = (int)height;
    *out_width = (int)width;
    *out_height = (int)height;

    // Allocate buffers
    size_t bufSize = width * height * 4;
    for (int i = 0; i < 2; i++) {
        cap->frame_buffer[i] = (unsigned char*)malloc(bufSize);
        if (!cap->frame_buffer[i]) {
            for (int j = 0; j < i; j++) free(cap->frame_buffer[j]);
            free(cap);
            return NULL;
        }
        memset(cap->frame_buffer[i], 0, bufSize);
    }
    cap->prev_frame = (unsigned char*)malloc(bufSize);

    // Create dispatch queue
    cap->queue = dispatch_queue_create("com.examguard.capture", DISPATCH_QUEUE_SERIAL);

    // Create display stream properties
    CFMutableDictionaryRef streamProperties = CFDictionaryCreateMutable(
        kCFAllocatorDefault, 0,
        &kCFTypeDictionaryKeyCallBacks,
        &kCFTypeDictionaryValueCallBacks
    );

    // Set pixel format (BGRA)
    int pixelFormat = 'BGRA';
    CFNumberRef pixelFormatNum = CFNumberCreate(kCFAllocatorDefault, kCFNumberIntType, &pixelFormat);
    CFDictionarySetValue(streamProperties, kCGDisplayStreamSourceRect,
        CGRectCreateDictionaryRepresentation(CGRectMake(0, 0, width, height)));

    // Request 8 FPS
    float fps = 8.0f;
    CFNumberRef fpsNum = CFNumberCreate(kCFAllocatorDefault, kCFNumberFloatType, &fps);
    CFDictionarySetValue(streamProperties, kCGDisplayStreamMinimumFrameTime, fpsNum);
    CFRelease(fpsNum);

    // Show cursor
    CFDictionarySetValue(streamProperties, kCGDisplayStreamShowCursor, kCFBooleanTrue);

    // Create the display stream
    cap->stream = CGDisplayStreamCreateWithDispatchQueue(
        displayID,
        width,
        height,
        'BGRA',
        streamProperties,
        cap->queue,
        ^(CGDisplayStreamFrameStatus status, uint64_t displayTime,
          IOSurfaceRef frameSurface, CGDisplayStreamUpdateRef updateRef) {
            cg_capture_callback(status, displayTime, frameSurface, updateRef, cap);
        }
    );

    CFRelease(streamProperties);
    CFRelease(pixelFormatNum);

    if (!cap->stream) {
        for (int i = 0; i < 2; i++) free(cap->frame_buffer[i]);
        free(cap->prev_frame);
        dispatch_release(cap->queue);
        free(cap);
        return NULL;
    }

    // Start the stream
    cap->running = 1;
    CGError err = CGDisplayStreamStart(cap->stream);
    if (err != kCGErrorSuccess) {
        CFRelease(cap->stream);
        for (int i = 0; i < 2; i++) free(cap->frame_buffer[i]);
        free(cap->prev_frame);
        dispatch_release(cap->queue);
        free(cap);
        return NULL;
    }

    return cap;
}

// Get current frame and detect dirty rectangles
// Returns number of dirty rects, -1 for full frame, 0 for no new frame
int cg_capture_frame(CGCapture *cap, unsigned char *rgba_out, int *dirty_rects, int max_rects) {
    if (!cap || !cap->running) return 0;

    pthread_mutex_lock(&cap->mutex);

    if (!cap->frame_ready) {
        pthread_mutex_unlock(&cap->mutex);
        return 0;
    }

    // Copy current frame
    int size = cap->frame_width * cap->frame_height * 4;
    memcpy(rgba_out, cap->frame_buffer[cap->current_buffer], size);

    // Detect dirty rectangles by comparing with previous frame
    int dirty_count = 0;
    int block_size = 64;
    int blocks_x = (cap->frame_width + block_size - 1) / block_size;
    int blocks_y = (cap->frame_height + block_size - 1) / block_size;
    int stride = cap->frame_width * 4;

    unsigned char *curr = cap->frame_buffer[cap->current_buffer];
    unsigned char *prev = cap->prev_frame;

    for (int by = 0; by < blocks_y && dirty_count < max_rects; by++) {
        for (int bx = 0; bx < blocks_x && dirty_count < max_rects; bx++) {
            int rx = bx * block_size;
            int ry = by * block_size;
            int rw = (rx + block_size > cap->frame_width) ? (cap->frame_width - rx) : block_size;
            int rh = (ry + block_size > cap->frame_height) ? (cap->frame_height - ry) : block_size;

            int changed = 0;
            for (int cy = ry; cy < ry + rh && !changed; cy += 8) {
                for (int cx = rx; cx < rx + rw && !changed; cx += 8) {
                    int idx = cy * stride + cx * 4;
                    if (memcmp(&curr[idx], &prev[idx], 4) != 0) {
                        changed = 1;
                    }
                }
            }

            if (changed) {
                dirty_rects[dirty_count * 4 + 0] = rx;
                dirty_rects[dirty_count * 4 + 1] = ry;
                dirty_rects[dirty_count * 4 + 2] = rw;
                dirty_rects[dirty_count * 4 + 3] = rh;
                dirty_count++;
            }
        }
    }

    // Save current frame as previous
    memcpy(cap->prev_frame, curr, size);
    cap->frame_ready = 0;

    pthread_mutex_unlock(&cap->mutex);

    // If no dirty rects detected, treat as full frame
    return dirty_count > 0 ? dirty_count : -1;
}

// Get dimensions
void cg_capture_get_size(CGCapture *cap, int *width, int *height) {
    if (cap) {
        pthread_mutex_lock(&cap->mutex);
        *width = cap->frame_width;
        *height = cap->frame_height;
        pthread_mutex_unlock(&cap->mutex);
    } else {
        *width = 0;
        *height = 0;
    }
}

// Cleanup
void cg_capture_destroy(CGCapture *cap) {
    if (!cap) return;

    cap->running = 0;

    if (cap->stream) {
        CGDisplayStreamStop(cap->stream);
        CFRelease(cap->stream);
    }

    if (cap->queue) {
        dispatch_release(cap->queue);
    }

    pthread_mutex_lock(&cap->mutex);
    for (int i = 0; i < 2; i++) {
        if (cap->frame_buffer[i]) free(cap->frame_buffer[i]);
    }
    if (cap->prev_frame) free(cap->prev_frame);
    pthread_mutex_unlock(&cap->mutex);

    pthread_mutex_destroy(&cap->mutex);
    free(cap);
}
*/
import "C"
import (
	"sync"
	"time"
	"unsafe"
)

const cgMaxDirtyRects = 32

// CGCapturer implements Capturer for macOS using CGDisplayStream.
type CGCapturer struct {
	cap       *C.CGCapture
	width     int
	height    int
	started   bool
	mu        sync.Mutex
	framePool *FramePool

	rgbaBuffer []byte
	dirtyBuf   []C.int

	keyFrameCounter int
}

func NewCGCapturer() *CGCapturer {
	return &CGCapturer{
		framePool: NewFramePool(),
		dirtyBuf:  make([]C.int, cgMaxDirtyRects*4),
	}
}

func (c *CGCapturer) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return ErrAlreadyStarted
	}

	var width, height C.int
	c.cap = C.cg_capture_init(&width, &height)
	if c.cap == nil {
		return ErrNoDisplay
	}

	c.width = int(width)
	c.height = int(height)
	c.started = true

	c.rgbaBuffer = make([]byte, c.width*c.height*4)

	time.Sleep(200 * time.Millisecond)

	return nil
}

func (c *CGCapturer) ReadFrame() (*FrameWithDirty, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return nil, ErrNotStarted
	}

	var width, height C.int
	C.cg_capture_get_size(c.cap, &width, &height)
	if int(width) != c.width || int(height) != c.height {
		c.width = int(width)
		c.height = int(height)
		c.rgbaBuffer = make([]byte, c.width*c.height*4)
	}

	numDirty := C.cg_capture_frame(
		c.cap,
		(*C.uchar)(unsafe.Pointer(&c.rgbaBuffer[0])),
		&c.dirtyBuf[0],
		cgMaxDirtyRects,
	)

	if numDirty == 0 {
		return nil, nil
	}

	frame := c.framePool.Get(c.width, c.height)
	copy(frame.Pix, c.rgbaBuffer)

	c.keyFrameCounter++
	forceKeyFrame := c.keyFrameCounter >= 40 // Every ~5 seconds at 8 FPS
	if forceKeyFrame {
		c.keyFrameCounter = 0
	}

	result := &FrameWithDirty{
		Frame:      frame,
		IsKeyFrame: numDirty == -1 || forceKeyFrame,
	}

	if numDirty > 0 && !result.IsKeyFrame {
		result.DirtyRects = make([]DirtyRect, numDirty)
		for i := 0; i < int(numDirty); i++ {
			result.DirtyRects[i] = DirtyRect{
				X: int(c.dirtyBuf[i*4+0]),
				Y: int(c.dirtyBuf[i*4+1]),
				W: int(c.dirtyBuf[i*4+2]),
				H: int(c.dirtyBuf[i*4+3]),
			}
		}
	}

	return result, nil
}

func (c *CGCapturer) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started && c.cap != nil {
		C.cg_capture_destroy(c.cap)
		c.cap = nil
		c.started = false
	}
}

func (c *CGCapturer) SupportsDirtyRects() bool {
	return false // Uses software comparison, not hardware-provided
}
