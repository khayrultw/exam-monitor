//go:build linux && wayland
// +build linux,wayland

package capture

/*
#cgo pkg-config: libpipewire-0.3 libspa-0.2

#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <unistd.h>
#include <fcntl.h>
#include <errno.h>
#include <pipewire/pipewire.h>
#include <spa/param/video/format-utils.h>
#include <spa/debug/types.h>
#include <spa/param/video/type-info.h>

// PipeWire capture state
typedef struct {
    struct pw_thread_loop *loop;
    struct pw_context *context;
    struct pw_core *core;
    struct pw_stream *stream;
    struct spa_hook stream_listener;

    // Frame buffer
    unsigned char *frame_data;
    int frame_width;
    int frame_height;
    int frame_stride;
    int frame_ready;
    int format; // SPA_VIDEO_FORMAT_*

    // Sync primitives
    pthread_mutex_t mutex;
    int running;
    int started;

    // Portal node id
    uint32_t node_id;
} PWCapture;

static void on_stream_param_changed(void *userdata, uint32_t id, const struct spa_pod *param) {
    PWCapture *cap = (PWCapture*)userdata;
    if (!param || id != SPA_PARAM_Format) return;

    struct spa_video_info info;
    if (spa_format_video_parse(param, &info) < 0) return;

    if (info.media_type != SPA_MEDIA_TYPE_video ||
        info.media_subtype != SPA_MEDIA_SUBTYPE_raw) return;

    cap->frame_width = info.info.raw.size.width;
    cap->frame_height = info.info.raw.size.height;
    cap->format = info.info.raw.format;

    // Calculate stride based on format
    int bpp = 4; // Assume BGRA/RGBA
    cap->frame_stride = cap->frame_width * bpp;

    // Allocate frame buffer
    pthread_mutex_lock(&cap->mutex);
    if (cap->frame_data) free(cap->frame_data);
    cap->frame_data = (unsigned char*)malloc(cap->frame_stride * cap->frame_height);
    pthread_mutex_unlock(&cap->mutex);
}

static void on_stream_process(void *userdata) {
    PWCapture *cap = (PWCapture*)userdata;
    struct pw_buffer *buf;
    struct spa_buffer *spa_buf;

    if (!cap->running) return;

    buf = pw_stream_dequeue_buffer(cap->stream);
    if (!buf) return;

    spa_buf = buf->buffer;

    if (spa_buf->datas[0].data) {
        pthread_mutex_lock(&cap->mutex);
        if (cap->frame_data && cap->frame_width > 0 && cap->frame_height > 0) {
            // Copy frame data
            int src_stride = spa_buf->datas[0].chunk->stride;
            if (src_stride == 0) src_stride = cap->frame_stride;

            unsigned char *src = (unsigned char*)spa_buf->datas[0].data;
            for (int y = 0; y < cap->frame_height; y++) {
                memcpy(cap->frame_data + y * cap->frame_stride,
                       src + y * src_stride,
                       cap->frame_stride);
            }
            cap->frame_ready = 1;
        }
        pthread_mutex_unlock(&cap->mutex);
    }

    pw_stream_queue_buffer(cap->stream, buf);
}

static void on_stream_state_changed(void *userdata, enum pw_stream_state old,
                                     enum pw_stream_state state, const char *error) {
    PWCapture *cap = (PWCapture*)userdata;
    if (state == PW_STREAM_STATE_STREAMING) {
        cap->started = 1;
    }
}

static const struct pw_stream_events stream_events = {
    PW_VERSION_STREAM_EVENTS,
    .param_changed = on_stream_param_changed,
    .process = on_stream_process,
    .state_changed = on_stream_state_changed,
};

// Initialize PipeWire capture with a node ID from portal
PWCapture* pw_capture_init(uint32_t node_id) {
    PWCapture *cap = (PWCapture*)calloc(1, sizeof(PWCapture));
    if (!cap) return NULL;

    pthread_mutex_init(&cap->mutex, NULL);
    cap->node_id = node_id;

    pw_init(NULL, NULL);

    cap->loop = pw_thread_loop_new("pw-capture", NULL);
    if (!cap->loop) {
        free(cap);
        return NULL;
    }

    cap->context = pw_context_new(pw_thread_loop_get_loop(cap->loop), NULL, 0);
    if (!cap->context) {
        pw_thread_loop_destroy(cap->loop);
        free(cap);
        return NULL;
    }

    if (pw_thread_loop_start(cap->loop) < 0) {
        pw_context_destroy(cap->context);
        pw_thread_loop_destroy(cap->loop);
        free(cap);
        return NULL;
    }

    pw_thread_loop_lock(cap->loop);

    cap->core = pw_context_connect(cap->context, NULL, 0);
    if (!cap->core) {
        pw_thread_loop_unlock(cap->loop);
        pw_thread_loop_stop(cap->loop);
        pw_context_destroy(cap->context);
        pw_thread_loop_destroy(cap->loop);
        free(cap);
        return NULL;
    }

    // Create stream
    struct pw_properties *props = pw_properties_new(
        PW_KEY_MEDIA_TYPE, "Video",
        PW_KEY_MEDIA_CATEGORY, "Capture",
        PW_KEY_MEDIA_ROLE, "Screen",
        NULL
    );

    cap->stream = pw_stream_new(cap->core, "screen-capture", props);
    if (!cap->stream) {
        pw_core_disconnect(cap->core);
        pw_thread_loop_unlock(cap->loop);
        pw_thread_loop_stop(cap->loop);
        pw_context_destroy(cap->context);
        pw_thread_loop_destroy(cap->loop);
        free(cap);
        return NULL;
    }

    pw_stream_add_listener(cap->stream, &cap->stream_listener, &stream_events, cap);

    // Build format
    uint8_t buffer[1024];
    struct spa_pod_builder b = SPA_POD_BUILDER_INIT(buffer, sizeof(buffer));

    const struct spa_pod *params[1];
    params[0] = spa_pod_builder_add_object(&b,
        SPA_TYPE_OBJECT_Format, SPA_PARAM_EnumFormat,
        SPA_FORMAT_mediaType, SPA_POD_Id(SPA_MEDIA_TYPE_video),
        SPA_FORMAT_mediaSubtype, SPA_POD_Id(SPA_MEDIA_SUBTYPE_raw),
        SPA_FORMAT_VIDEO_format, SPA_POD_CHOICE_ENUM_Id(4,
            SPA_VIDEO_FORMAT_BGRA,
            SPA_VIDEO_FORMAT_BGRx,
            SPA_VIDEO_FORMAT_RGBA,
            SPA_VIDEO_FORMAT_RGBx),
        SPA_FORMAT_VIDEO_size, SPA_POD_CHOICE_RANGE_Rectangle(
            &SPA_RECTANGLE(1920, 1080),
            &SPA_RECTANGLE(1, 1),
            &SPA_RECTANGLE(8192, 8192)),
        SPA_FORMAT_VIDEO_framerate, SPA_POD_CHOICE_RANGE_Fraction(
            &SPA_FRACTION(8, 1),
            &SPA_FRACTION(0, 1),
            &SPA_FRACTION(60, 1)));

    // Connect with node_id target
    char target[64];
    snprintf(target, sizeof(target), "%u", node_id);

    pw_stream_connect(cap->stream,
        PW_DIRECTION_INPUT,
        node_id,
        PW_STREAM_FLAG_AUTOCONNECT | PW_STREAM_FLAG_MAP_BUFFERS,
        params, 1);

    cap->running = 1;

    pw_thread_loop_unlock(cap->loop);

    return cap;
}

// Get frame dimensions
void pw_capture_get_size(PWCapture *cap, int *width, int *height) {
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

// Copy frame to RGBA buffer, returns 1 if new frame available
int pw_capture_frame(PWCapture *cap, unsigned char *rgba_out, int *width, int *height) {
    if (!cap) return 0;

    pthread_mutex_lock(&cap->mutex);

    if (!cap->frame_ready || !cap->frame_data) {
        pthread_mutex_unlock(&cap->mutex);
        return 0;
    }

    *width = cap->frame_width;
    *height = cap->frame_height;

    // Convert to RGBA based on source format
    int pixels = cap->frame_width * cap->frame_height;
    unsigned char *src = cap->frame_data;

    if (cap->format == SPA_VIDEO_FORMAT_BGRA || cap->format == SPA_VIDEO_FORMAT_BGRx) {
        // BGRA to RGBA
        for (int i = 0; i < pixels; i++) {
            rgba_out[i*4 + 0] = src[i*4 + 2]; // R
            rgba_out[i*4 + 1] = src[i*4 + 1]; // G
            rgba_out[i*4 + 2] = src[i*4 + 0]; // B
            rgba_out[i*4 + 3] = 255;          // A
        }
    } else {
        // RGBA/RGBx - just copy with alpha fix
        for (int i = 0; i < pixels; i++) {
            rgba_out[i*4 + 0] = src[i*4 + 0];
            rgba_out[i*4 + 1] = src[i*4 + 1];
            rgba_out[i*4 + 2] = src[i*4 + 2];
            rgba_out[i*4 + 3] = 255;
        }
    }

    cap->frame_ready = 0;
    pthread_mutex_unlock(&cap->mutex);

    return 1;
}

// Check if stream is started
int pw_capture_is_started(PWCapture *cap) {
    return cap ? cap->started : 0;
}

// Cleanup
void pw_capture_destroy(PWCapture *cap) {
    if (!cap) return;

    cap->running = 0;

    if (cap->loop) {
        pw_thread_loop_lock(cap->loop);

        if (cap->stream) {
            spa_hook_remove(&cap->stream_listener);
            pw_stream_disconnect(cap->stream);
            pw_stream_destroy(cap->stream);
        }

        if (cap->core) {
            pw_core_disconnect(cap->core);
        }

        pw_thread_loop_unlock(cap->loop);
        pw_thread_loop_stop(cap->loop);

        if (cap->context) {
            pw_context_destroy(cap->context);
        }

        pw_thread_loop_destroy(cap->loop);
    }

    pthread_mutex_lock(&cap->mutex);
    if (cap->frame_data) {
        free(cap->frame_data);
    }
    pthread_mutex_unlock(&cap->mutex);
    pthread_mutex_destroy(&cap->mutex);

    pw_deinit();
    free(cap);
}
*/
import "C"
import (
	"sync"
	"time"
	"unsafe"
)

// WaylandCapturer implements Capturer for Linux Wayland using PipeWire.
// Note: Requires user to grant screen capture permission via xdg-desktop-portal.
type WaylandCapturer struct {
	cap        *C.PWCapture
	width      int
	height     int
	started    bool
	mu         sync.Mutex
	framePool  *FramePool
	rgbaBuffer []byte
	nodeID     uint32

	prevFrame     []byte
	keyFrameCount int
}

// NewWaylandCapturer creates a new Wayland screen capturer.
// nodeID is the PipeWire node ID obtained from xdg-desktop-portal.
func NewWaylandCapturer(nodeID uint32) *WaylandCapturer {
	return &WaylandCapturer{
		framePool: NewFramePool(),
		nodeID:    nodeID,
	}
}

func (c *WaylandCapturer) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return ErrAlreadyStarted
	}

	c.cap = C.pw_capture_init(C.uint32_t(c.nodeID))
	if c.cap == nil {
		return ErrNoDisplay
	}

	timeout := time.After(5 * time.Second)
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-timeout:
			C.pw_capture_destroy(c.cap)
			c.cap = nil
			return ErrCaptureFailed
		case <-tick.C:
			if C.pw_capture_is_started(c.cap) != 0 {
				var w, h C.int
				C.pw_capture_get_size(c.cap, &w, &h)
				c.width = int(w)
				c.height = int(h)
				if c.width > 0 && c.height > 0 {
					c.rgbaBuffer = make([]byte, c.width*c.height*4)
					c.prevFrame = make([]byte, c.width*c.height*4)
					c.started = true
					return nil
				}
			}
		}
	}
}

func (c *WaylandCapturer) ReadFrame() (*FrameWithDirty, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return nil, ErrNotStarted
	}

	var w, h C.int
	hasFrame := C.pw_capture_frame(
		c.cap,
		(*C.uchar)(unsafe.Pointer(&c.rgbaBuffer[0])),
		&w, &h,
	)

	if hasFrame == 0 {
		return nil, nil
	}

	newWidth := int(w)
	newHeight := int(h)
	if newWidth != c.width || newHeight != c.height {
		c.width = newWidth
		c.height = newHeight
		c.rgbaBuffer = make([]byte, c.width*c.height*4)
		c.prevFrame = make([]byte, c.width*c.height*4)
	}

	frame := c.framePool.Get(c.width, c.height)
	copy(frame.Pix, c.rgbaBuffer)

	c.keyFrameCount++
	isKeyFrame := c.keyFrameCount >= 40 // Every ~5 seconds at 8 FPS
	if isKeyFrame {
		c.keyFrameCount = 0
	}

	result := &FrameWithDirty{
		Frame:      frame,
		IsKeyFrame: isKeyFrame,
	}

	if !isKeyFrame && len(c.prevFrame) == len(c.rgbaBuffer) {
		dirtyRects := detectDirtyRects(c.rgbaBuffer, c.prevFrame, c.width, c.height, 64)
		result.DirtyRects = dirtyRects
		if len(dirtyRects) == 0 {
			c.framePool.Put(frame)
			return nil, nil
		}
	}

	copy(c.prevFrame, c.rgbaBuffer)

	return result, nil
}

func (c *WaylandCapturer) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started && c.cap != nil {
		C.pw_capture_destroy(c.cap)
		c.cap = nil
		c.started = false
	}
}

func (c *WaylandCapturer) SupportsDirtyRects() bool {
	return false
}

func detectDirtyRects(current, prev []byte, width, height, blockSize int) []DirtyRect {
	var rects []DirtyRect

	blocksX := (width + blockSize - 1) / blockSize
	blocksY := (height + blockSize - 1) / blockSize
	stride := width * 4

	for by := 0; by < blocksY; by++ {
		for bx := 0; bx < blocksX; bx++ {
			rx := bx * blockSize
			ry := by * blockSize
			rw := blockSize
			rh := blockSize
			if rx+rw > width {
				rw = width - rx
			}
			if ry+rh > height {
				rh = height - ry
			}

			changed := false
			for cy := ry; cy < ry+rh && !changed; cy += 8 {
				for cx := rx; cx < rx+rw && !changed; cx += 8 {
					idx := cy*stride + cx*4
					if idx+4 <= len(current) && idx+4 <= len(prev) {
						if current[idx] != prev[idx] ||
							current[idx+1] != prev[idx+1] ||
							current[idx+2] != prev[idx+2] {
							changed = true
						}
					}
				}
			}

			if changed {
				rects = append(rects, DirtyRect{X: rx, Y: ry, W: rw, H: rh})
			}
		}
	}

	return rects
}
