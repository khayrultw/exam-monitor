//go:build darwin && cgo
// +build darwin,cgo

#ifndef SCREENCAPTUREKIT_DARWIN_H
#define SCREENCAPTUREKIT_DARWIN_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct SCKCapture SCKCapture;

// Initialize capture (returns NULL on failure)
SCKCapture* sck_capture_init(int *out_width, int *out_height, char **error);

// Start capturing
int sck_capture_start(SCKCapture *cap);

// Get frame (returns 1 if frame available, 0 if not, -1 on error)
int sck_capture_get_frame(SCKCapture *cap, uint8_t *rgba_out);

// Get dimensions
void sck_capture_get_size(SCKCapture *cap, int *width, int *height);

// Stop capturing
void sck_capture_stop(SCKCapture *cap);

// Cleanup
void sck_capture_destroy(SCKCapture *cap);

#ifdef __cplusplus
}
#endif

#endif // SCREENCAPTUREKIT_DARWIN_H