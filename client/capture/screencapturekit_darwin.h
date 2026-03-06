//go:build darwin && cgo
// +build darwin,cgo

#ifndef SCREENCAPTUREKIT_DARWIN_H
#define SCREENCAPTUREKIT_DARWIN_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct SCKCapture SCKCapture;

// Check screen recording permission (returns 1 if granted, 0 if not).
// If not granted, prompts the user via system dialog.
int sck_check_screen_recording_permission();

// Initialize capture (returns NULL on failure)
SCKCapture* sck_capture_init(int *out_width, int *out_height, char **error);

// Start capturing
int sck_capture_start(SCKCapture *cap);

// Get frame atomically with dimensions.
// Returns 1 if frame available, 0 if not, -1 on error, -2 if buffer too small.
// When returning -2, out_width and out_height contain the required dimensions
// and the frame is preserved for retry after buffer reallocation.
int sck_capture_get_frame(SCKCapture *cap, uint8_t *rgba_out, int buf_capacity,
                          int *out_width, int *out_height);

// Stop capturing
void sck_capture_stop(SCKCapture *cap);

// Cleanup
void sck_capture_destroy(SCKCapture *cap);

#ifdef __cplusplus
}
#endif

#endif // SCREENCAPTUREKIT_DARWIN_H
