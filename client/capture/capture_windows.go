//go:build windows
// +build windows

package capture

/*
#cgo LDFLAGS: -ld3d11 -ldxgi -lole32 -luuid

#include <stdlib.h>
#include <string.h>
#include <windows.h>
#include <d3d11.h>
#include <dxgi1_2.h>

// DXGI Desktop Duplication capture state
typedef struct {
    ID3D11Device *device;
    ID3D11DeviceContext *context;
    IDXGIOutputDuplication *duplication;
    ID3D11Texture2D *staging_texture;

    int width;
    int height;
    int initialized;

    // Previous frame for dirty rect comparison
    unsigned char *prev_frame;
    int prev_frame_size;

    // Dirty rects from DXGI
    RECT *dirty_rects;
    int dirty_rect_count;
    int dirty_rect_capacity;
} DXGICapture;

// Initialize DXGI Desktop Duplication
DXGICapture* dxgi_capture_init(int *out_width, int *out_height) {
    DXGICapture *cap = (DXGICapture*)calloc(1, sizeof(DXGICapture));
    if (!cap) return NULL;

    HRESULT hr;

    // Create D3D11 device
    D3D_FEATURE_LEVEL featureLevels[] = { D3D_FEATURE_LEVEL_11_0, D3D_FEATURE_LEVEL_10_1 };
    D3D_FEATURE_LEVEL featureLevel;

    hr = D3D11CreateDevice(
        NULL,
        D3D_DRIVER_TYPE_HARDWARE,
        NULL,
        0,
        featureLevels,
        2,
        D3D11_SDK_VERSION,
        &cap->device,
        &featureLevel,
        &cap->context
    );

    if (FAILED(hr)) {
        free(cap);
        return NULL;
    }

    // Get DXGI device
    IDXGIDevice *dxgiDevice;
    hr = cap->device->lpVtbl->QueryInterface(cap->device, &IID_IDXGIDevice, (void**)&dxgiDevice);
    if (FAILED(hr)) {
        cap->device->lpVtbl->Release(cap->device);
        free(cap);
        return NULL;
    }

    // Get adapter
    IDXGIAdapter *adapter;
    hr = dxgiDevice->lpVtbl->GetParent(dxgiDevice, &IID_IDXGIAdapter, (void**)&adapter);
    dxgiDevice->lpVtbl->Release(dxgiDevice);

    if (FAILED(hr)) {
        cap->device->lpVtbl->Release(cap->device);
        free(cap);
        return NULL;
    }

    // Get primary output
    IDXGIOutput *output;
    hr = adapter->lpVtbl->EnumOutputs(adapter, 0, &output);
    adapter->lpVtbl->Release(adapter);

    if (FAILED(hr)) {
        cap->device->lpVtbl->Release(cap->device);
        free(cap);
        return NULL;
    }

    // Get output description for dimensions
    DXGI_OUTPUT_DESC outputDesc;
    output->lpVtbl->GetDesc(output, &outputDesc);
    cap->width = outputDesc.DesktopCoordinates.right - outputDesc.DesktopCoordinates.left;
    cap->height = outputDesc.DesktopCoordinates.bottom - outputDesc.DesktopCoordinates.top;
    *out_width = cap->width;
    *out_height = cap->height;

    // Get Output1 interface for duplication
    IDXGIOutput1 *output1;
    hr = output->lpVtbl->QueryInterface(output, &IID_IDXGIOutput1, (void**)&output1);
    output->lpVtbl->Release(output);

    if (FAILED(hr)) {
        cap->device->lpVtbl->Release(cap->device);
        free(cap);
        return NULL;
    }

    // Create desktop duplication
    hr = output1->lpVtbl->DuplicateOutput(output1, (IUnknown*)cap->device, &cap->duplication);
    output1->lpVtbl->Release(output1);

    if (FAILED(hr)) {
        cap->context->lpVtbl->Release(cap->context);
        cap->device->lpVtbl->Release(cap->device);
        free(cap);
        return NULL;
    }

    // Create staging texture for CPU access
    D3D11_TEXTURE2D_DESC texDesc;
    ZeroMemory(&texDesc, sizeof(texDesc));
    texDesc.Width = cap->width;
    texDesc.Height = cap->height;
    texDesc.MipLevels = 1;
    texDesc.ArraySize = 1;
    texDesc.Format = DXGI_FORMAT_B8G8R8A8_UNORM;
    texDesc.SampleDesc.Count = 1;
    texDesc.Usage = D3D11_USAGE_STAGING;
    texDesc.CPUAccessFlags = D3D11_CPU_ACCESS_READ;

    hr = cap->device->lpVtbl->CreateTexture2D(cap->device, &texDesc, NULL, &cap->staging_texture);
    if (FAILED(hr)) {
        cap->duplication->lpVtbl->Release(cap->duplication);
        cap->context->lpVtbl->Release(cap->context);
        cap->device->lpVtbl->Release(cap->device);
        free(cap);
        return NULL;
    }

    // Allocate dirty rect storage
    cap->dirty_rect_capacity = 64;
    cap->dirty_rects = (RECT*)malloc(sizeof(RECT) * cap->dirty_rect_capacity);

    // Allocate previous frame buffer
    cap->prev_frame_size = cap->width * cap->height * 4;
    cap->prev_frame = (unsigned char*)malloc(cap->prev_frame_size);

    cap->initialized = 1;
    return cap;
}

// Capture frame with dirty rectangles
// Returns number of dirty rects, -1 for full frame, -2 for timeout/no change
int dxgi_capture_frame(DXGICapture *cap, unsigned char *rgba_out, int *dirty_rects_out, int max_rects) {
    if (!cap || !cap->initialized) return -2;

    HRESULT hr;
    IDXGIResource *resource = NULL;
    DXGI_OUTDUPL_FRAME_INFO frameInfo;

    // Acquire next frame with short timeout
    hr = cap->duplication->lpVtbl->AcquireNextFrame(cap->duplication, 100, &frameInfo, &resource);

    if (hr == DXGI_ERROR_WAIT_TIMEOUT) {
        return -2; // No new frame
    }

    if (hr == DXGI_ERROR_ACCESS_LOST) {
        // Need to recreate duplication (e.g., after resolution change)
        cap->initialized = 0;
        return -2;
    }

    if (FAILED(hr)) {
        return -2;
    }

    int dirty_count = 0;
    int full_frame = 1;

    // Get dirty rectangles from DXGI
    if (frameInfo.TotalMetadataBufferSize > 0) {
        UINT bufferSize = 0;

        // First call to get required size
        cap->duplication->lpVtbl->GetFrameDirtyRects(cap->duplication, 0, NULL, &bufferSize);

        if (bufferSize > 0) {
            UINT maxRects = bufferSize / sizeof(RECT);
            if (maxRects > (UINT)cap->dirty_rect_capacity) {
                cap->dirty_rects = (RECT*)realloc(cap->dirty_rects, bufferSize);
                cap->dirty_rect_capacity = maxRects;
            }

            hr = cap->duplication->lpVtbl->GetFrameDirtyRects(
                cap->duplication, bufferSize, cap->dirty_rects, &bufferSize
            );

            if (SUCCEEDED(hr)) {
                dirty_count = bufferSize / sizeof(RECT);
                if (dirty_count > max_rects) dirty_count = max_rects;

                for (int i = 0; i < dirty_count; i++) {
                    dirty_rects_out[i * 4 + 0] = cap->dirty_rects[i].left;
                    dirty_rects_out[i * 4 + 1] = cap->dirty_rects[i].top;
                    dirty_rects_out[i * 4 + 2] = cap->dirty_rects[i].right - cap->dirty_rects[i].left;
                    dirty_rects_out[i * 4 + 3] = cap->dirty_rects[i].bottom - cap->dirty_rects[i].top;
                }
                full_frame = 0;
            }
        }
    }

    // Get texture from resource
    ID3D11Texture2D *texture;
    hr = resource->lpVtbl->QueryInterface(resource, &IID_ID3D11Texture2D, (void**)&texture);
    resource->lpVtbl->Release(resource);

    if (FAILED(hr)) {
        cap->duplication->lpVtbl->ReleaseFrame(cap->duplication);
        return -2;
    }

    // Copy to staging texture
    cap->context->lpVtbl->CopyResource(cap->context,
        (ID3D11Resource*)cap->staging_texture,
        (ID3D11Resource*)texture);
    texture->lpVtbl->Release(texture);

    // Map staging texture
    D3D11_MAPPED_SUBRESOURCE mapped;
    hr = cap->context->lpVtbl->Map(cap->context,
        (ID3D11Resource*)cap->staging_texture,
        0, D3D11_MAP_READ, 0, &mapped);

    if (FAILED(hr)) {
        cap->duplication->lpVtbl->ReleaseFrame(cap->duplication);
        return -2;
    }

    // Convert BGRA to RGBA
    unsigned char *src = (unsigned char*)mapped.pData;
    int dst_stride = cap->width * 4;

    for (int y = 0; y < cap->height; y++) {
        for (int x = 0; x < cap->width; x++) {
            int src_idx = y * mapped.RowPitch + x * 4;
            int dst_idx = y * dst_stride + x * 4;

            rgba_out[dst_idx + 0] = src[src_idx + 2]; // R
            rgba_out[dst_idx + 1] = src[src_idx + 1]; // G
            rgba_out[dst_idx + 2] = src[src_idx + 0]; // B
            rgba_out[dst_idx + 3] = 255;              // A
        }
    }

    cap->context->lpVtbl->Unmap(cap->context, (ID3D11Resource*)cap->staging_texture, 0);
    cap->duplication->lpVtbl->ReleaseFrame(cap->duplication);

    return full_frame ? -1 : dirty_count;
}

// Get dimensions
void dxgi_capture_get_size(DXGICapture *cap, int *width, int *height) {
    if (cap) {
        *width = cap->width;
        *height = cap->height;
    } else {
        *width = 0;
        *height = 0;
    }
}

// Check if initialized
int dxgi_capture_is_valid(DXGICapture *cap) {
    return cap && cap->initialized;
}

// Cleanup
void dxgi_capture_destroy(DXGICapture *cap) {
    if (!cap) return;

    if (cap->dirty_rects) free(cap->dirty_rects);
    if (cap->prev_frame) free(cap->prev_frame);

    if (cap->staging_texture) cap->staging_texture->lpVtbl->Release(cap->staging_texture);
    if (cap->duplication) cap->duplication->lpVtbl->Release(cap->duplication);
    if (cap->context) cap->context->lpVtbl->Release(cap->context);
    if (cap->device) cap->device->lpVtbl->Release(cap->device);

    free(cap);
}
*/
import "C"
import (
	"sync"
	"unsafe"
)

const dxgiMaxDirtyRects = 32

type DXGICapturer struct {
	cap       *C.DXGICapture
	width     int
	height    int
	started   bool
	mu        sync.Mutex
	framePool *FramePool

	rgbaBuffer []byte
	dirtyBuf   []C.int

	keyFrameCounter int
}

func NewDXGICapturer() *DXGICapturer {
	return &DXGICapturer{
		framePool: NewFramePool(),
		dirtyBuf:  make([]C.int, dxgiMaxDirtyRects*4),
	}
}

func (c *DXGICapturer) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return ErrAlreadyStarted
	}

	var width, height C.int
	c.cap = C.dxgi_capture_init(&width, &height)
	if c.cap == nil {
		return ErrNoDisplay
	}

	c.width = int(width)
	c.height = int(height)
	c.started = true

	c.rgbaBuffer = make([]byte, c.width*c.height*4)

	return nil
}

func (c *DXGICapturer) ReadFrame() (*FrameWithDirty, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return nil, ErrNotStarted
	}

	if C.dxgi_capture_is_valid(c.cap) == 0 {
		C.dxgi_capture_destroy(c.cap)
		var width, height C.int
		c.cap = C.dxgi_capture_init(&width, &height)
		if c.cap == nil {
			c.started = false
			return nil, ErrCaptureFailed
		}
		c.width = int(width)
		c.height = int(height)
		c.rgbaBuffer = make([]byte, c.width*c.height*4)
	}

	numDirty := C.dxgi_capture_frame(
		c.cap,
		(*C.uchar)(unsafe.Pointer(&c.rgbaBuffer[0])),
		&c.dirtyBuf[0],
		dxgiMaxDirtyRects,
	)

	if numDirty == -2 {
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

func (c *DXGICapturer) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started && c.cap != nil {
		C.dxgi_capture_destroy(c.cap)
		c.cap = nil
		c.started = false
	}
}

func (c *DXGICapturer) SupportsDirtyRects() bool {
	return true
}
