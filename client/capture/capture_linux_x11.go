//go:build linux && !wayland
// +build linux,!wayland

package capture

/*
#cgo LDFLAGS: -lX11 -lXext -lXdamage

#include <stdlib.h>
#include <string.h>
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <X11/extensions/XShm.h>
#include <X11/extensions/Xdamage.h>
#include <sys/shm.h>
#include <sys/ipc.h>

// X11Capture holds X11 capture state
typedef struct {
    Display *display;
    Window root;
    int screen;
    int width;
    int height;
    int depth;

    // XShm for fast capture
    XShmSegmentInfo shminfo;
    XImage *image;
    int shm_attached;

    // XDamage for dirty rectangles
    Damage damage;
    int damage_event_base;
    int damage_error_base;
    int damage_supported;

    // Previous frame for comparison fallback
    unsigned char *prev_frame;
    int prev_frame_size;
} X11Capture;

// Initialize X11 capture
X11Capture* x11_capture_init(int *out_width, int *out_height) {
    X11Capture *cap = (X11Capture*)calloc(1, sizeof(X11Capture));
    if (!cap) return NULL;

    cap->display = XOpenDisplay(NULL);
    if (!cap->display) {
        free(cap);
        return NULL;
    }

    cap->screen = DefaultScreen(cap->display);
    cap->root = RootWindow(cap->display, cap->screen);
    cap->width = DisplayWidth(cap->display, cap->screen);
    cap->height = DisplayHeight(cap->display, cap->screen);
    cap->depth = DefaultDepth(cap->display, cap->screen);

    *out_width = cap->width;
    *out_height = cap->height;

    // Check XDamage support
    cap->damage_supported = XDamageQueryExtension(
        cap->display,
        &cap->damage_event_base,
        &cap->damage_error_base
    );

    if (cap->damage_supported) {
        cap->damage = XDamageCreate(cap->display, cap->root, XDamageReportRawRectangles);
    }

    // Check XShm support
    if (!XShmQueryExtension(cap->display)) {
        if (cap->damage_supported) {
            XDamageDestroy(cap->display, cap->damage);
        }
        XCloseDisplay(cap->display);
        free(cap);
        return NULL;
    }

    // Create shared memory image
    cap->image = XShmCreateImage(
        cap->display,
        DefaultVisual(cap->display, cap->screen),
        cap->depth,
        ZPixmap,
        NULL,
        &cap->shminfo,
        cap->width,
        cap->height
    );

    if (!cap->image) {
        if (cap->damage_supported) {
            XDamageDestroy(cap->display, cap->damage);
        }
        XCloseDisplay(cap->display);
        free(cap);
        return NULL;
    }

    // Allocate shared memory
    cap->shminfo.shmid = shmget(
        IPC_PRIVATE,
        cap->image->bytes_per_line * cap->image->height,
        IPC_CREAT | 0777
    );

    if (cap->shminfo.shmid < 0) {
        XDestroyImage(cap->image);
        if (cap->damage_supported) {
            XDamageDestroy(cap->display, cap->damage);
        }
        XCloseDisplay(cap->display);
        free(cap);
        return NULL;
    }

    cap->shminfo.shmaddr = cap->image->data = (char*)shmat(cap->shminfo.shmid, NULL, 0);
    cap->shminfo.readOnly = False;

    if (!XShmAttach(cap->display, &cap->shminfo)) {
        shmdt(cap->shminfo.shmaddr);
        shmctl(cap->shminfo.shmid, IPC_RMID, NULL);
        XDestroyImage(cap->image);
        if (cap->damage_supported) {
            XDamageDestroy(cap->display, cap->damage);
        }
        XCloseDisplay(cap->display);
        free(cap);
        return NULL;
    }

    cap->shm_attached = 1;

    // Mark shm for removal after detach
    shmctl(cap->shminfo.shmid, IPC_RMID, NULL);

    // Allocate previous frame buffer for dirty rect fallback
    cap->prev_frame_size = cap->width * cap->height * 4;
    cap->prev_frame = (unsigned char*)malloc(cap->prev_frame_size);

    return cap;
}

// Capture frame and detect dirty rectangles
// dirty_rects: output array of 4 ints per rect (x, y, w, h), max 32 rects
// Returns number of dirty rects, or -1 for full frame
int x11_capture_frame(X11Capture *cap, unsigned char *rgba_out, int *dirty_rects, int max_rects) {
    if (!cap || !cap->display) return -1;

    int dirty_count = 0;
    int full_frame = 1;

    // Process XDamage events if supported
    if (cap->damage_supported) {
        XEvent event;
        XRectangle damage_bounds = {0, 0, 0, 0};
        int has_damage = 0;

        while (XCheckTypedEvent(cap->display, cap->damage_event_base + XDamageNotify, &event)) {
            XDamageNotifyEvent *dev = (XDamageNotifyEvent*)&event;

            if (dirty_count < max_rects) {
                dirty_rects[dirty_count * 4 + 0] = dev->area.x;
                dirty_rects[dirty_count * 4 + 1] = dev->area.y;
                dirty_rects[dirty_count * 4 + 2] = dev->area.width;
                dirty_rects[dirty_count * 4 + 3] = dev->area.height;
                dirty_count++;
            }
            has_damage = 1;
        }

        if (has_damage) {
            XDamageSubtract(cap->display, cap->damage, None, None);
            full_frame = 0;
        }
    }

    // Capture via XShm
    if (!XShmGetImage(cap->display, cap->root, cap->image, 0, 0, AllPlanes)) {
        return -1;
    }

    // Convert to RGBA based on X server byte order
    // XImage bits_per_pixel tells us bytes per pixel (usually 32 for TrueColor)
    // XImage byte_order tells us endianness (LSBFirst or MSBFirst)
    int x, y;
    unsigned char *src = (unsigned char*)cap->image->data;
    int src_stride = cap->image->bytes_per_line;
    int dst_stride = cap->width * 4;
    int bytes_per_pixel = cap->image->bits_per_pixel / 8;

    // Get visual info to determine actual color positions
    unsigned long red_mask = cap->image->red_mask;
    unsigned long green_mask = cap->image->green_mask;
    unsigned long blue_mask = cap->image->blue_mask;

    // Calculate bit shifts from masks
    int red_shift = 0, green_shift = 0, blue_shift = 0;
    if (red_mask) while (!(red_mask & (1 << red_shift))) red_shift++;
    if (green_mask) while (!(green_mask & (1 << green_shift))) green_shift++;
    if (blue_mask) while (!(blue_mask & (1 << blue_shift))) blue_shift++;

    for (y = 0; y < cap->height; y++) {
        for (x = 0; x < cap->width; x++) {
            int src_idx = y * src_stride + x * bytes_per_pixel;
            int dst_idx = y * dst_stride + x * 4;

            // Read pixel value based on byte order
            unsigned long pixel = 0;
            if (cap->image->byte_order == LSBFirst) {
                for (int b = 0; b < bytes_per_pixel; b++) {
                    pixel |= ((unsigned long)src[src_idx + b]) << (b * 8);
                }
            } else {
                for (int b = 0; b < bytes_per_pixel; b++) {
                    pixel |= ((unsigned long)src[src_idx + b]) << ((bytes_per_pixel - 1 - b) * 8);
                }
            }

            // Extract RGB using masks and shifts
            unsigned char r = (pixel & red_mask) >> red_shift;
            unsigned char g = (pixel & green_mask) >> green_shift;
            unsigned char b = (pixel & blue_mask) >> blue_shift;

            rgba_out[dst_idx + 0] = r;
            rgba_out[dst_idx + 1] = g;
            rgba_out[dst_idx + 2] = b;
            rgba_out[dst_idx + 3] = 255;
        }
    }

    // If XDamage not supported, do pixel comparison for dirty rects
    if (!cap->damage_supported && cap->prev_frame && dirty_count == 0) {
        // Simple grid-based dirty detection (16x16 blocks)
        int block_size = 64;
        int blocks_x = (cap->width + block_size - 1) / block_size;
        int blocks_y = (cap->height + block_size - 1) / block_size;

        for (int by = 0; by < blocks_y && dirty_count < max_rects; by++) {
            for (int bx = 0; bx < blocks_x && dirty_count < max_rects; bx++) {
                int rx = bx * block_size;
                int ry = by * block_size;
                int rw = (rx + block_size > cap->width) ? (cap->width - rx) : block_size;
                int rh = (ry + block_size > cap->height) ? (cap->height - ry) : block_size;

                int changed = 0;
                for (int cy = ry; cy < ry + rh && !changed; cy += 8) {
                    for (int cx = rx; cx < rx + rw && !changed; cx += 8) {
                        int idx = cy * dst_stride + cx * 4;
                        if (memcmp(&rgba_out[idx], &cap->prev_frame[idx], 4) != 0) {
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
                    full_frame = 0;
                }
            }
        }

        // Copy current frame to prev
        memcpy(cap->prev_frame, rgba_out, cap->prev_frame_size);
    }

    return full_frame ? -1 : dirty_count;
}

// Cleanup
void x11_capture_destroy(X11Capture *cap) {
    if (!cap) return;

    if (cap->prev_frame) {
        free(cap->prev_frame);
    }

    if (cap->damage_supported && cap->damage) {
        XDamageDestroy(cap->display, cap->damage);
    }

    if (cap->shm_attached) {
        XShmDetach(cap->display, &cap->shminfo);
        shmdt(cap->shminfo.shmaddr);
    }

    if (cap->image) {
        // Don't XDestroyImage as it would try to free shm data
        cap->image->data = NULL;
        XDestroyImage(cap->image);
    }

    if (cap->display) {
        XCloseDisplay(cap->display);
    }

    free(cap);
}

// Check if XDamage is supported
int x11_capture_has_damage(X11Capture *cap) {
    return cap ? cap->damage_supported : 0;
}
*/
import "C"
import (
	"sync"
	"unsafe"
)

const maxDirtyRects = 32

type X11Capturer struct {
	cap       *C.X11Capture
	width     int
	height    int
	started   bool
	mu        sync.Mutex
	framePool *FramePool

	rgbaBuffer []byte
	dirtyBuf   []C.int
}

func NewX11Capturer() *X11Capturer {
	return &X11Capturer{
		framePool: NewFramePool(),
		dirtyBuf:  make([]C.int, maxDirtyRects*4),
	}
}

func (c *X11Capturer) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return ErrAlreadyStarted
	}

	var width, height C.int
	c.cap = C.x11_capture_init(&width, &height)
	if c.cap == nil {
		return ErrNoDisplay
	}

	c.width = int(width)
	c.height = int(height)
	c.started = true

	c.rgbaBuffer = make([]byte, c.width*c.height*4)

	return nil
}

func (c *X11Capturer) ReadFrame() (*FrameWithDirty, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return nil, ErrNotStarted
	}

	numDirty := C.x11_capture_frame(
		c.cap,
		(*C.uchar)(unsafe.Pointer(&c.rgbaBuffer[0])),
		&c.dirtyBuf[0],
		maxDirtyRects,
	)

	if numDirty < -1 {
		return nil, ErrCaptureFailed
	}

	frame := c.framePool.Get(c.width, c.height)
	copy(frame.Pix, c.rgbaBuffer)

	result := &FrameWithDirty{
		Frame:      frame,
		IsKeyFrame: numDirty == -1,
	}

	if numDirty > 0 {
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

func (c *X11Capturer) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started && c.cap != nil {
		C.x11_capture_destroy(c.cap)
		c.cap = nil
		c.started = false
	}
}

func (c *X11Capturer) SupportsDirtyRects() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cap == nil {
		return false
	}
	return C.x11_capture_has_damage(c.cap) != 0
}
