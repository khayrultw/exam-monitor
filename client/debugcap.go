// +build ignore

package main

/*
#cgo LDFLAGS: -lX11 -lXext -lXdamage

#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <X11/extensions/XShm.h>
#include <sys/shm.h>
#include <sys/ipc.h>

void debug_capture() {
    Display *display = XOpenDisplay(NULL);
    if (!display) {
        printf("Failed to open display\n");
        return;
    }

    int screen = DefaultScreen(display);
    Window root = RootWindow(display, screen);
    int width = DisplayWidth(display, screen);
    int height = DisplayHeight(display, screen);
    int depth = DefaultDepth(display, screen);

    printf("Screen: %dx%d, depth=%d\n", width, height, depth);

    // Get visual info
    Visual *visual = DefaultVisual(display, screen);
    printf("Visual: red_mask=0x%lx green_mask=0x%lx blue_mask=0x%lx\n",
           visual->red_mask, visual->green_mask, visual->blue_mask);

    // Try XGetImage first (slower but more reliable)
    XImage *img = XGetImage(display, root, 0, 0, 100, 100, AllPlanes, ZPixmap);
    if (!img) {
        printf("XGetImage failed\n");
        XCloseDisplay(display);
        return;
    }

    printf("XImage: bits_per_pixel=%d, byte_order=%s, depth=%d\n",
           img->bits_per_pixel,
           img->byte_order == LSBFirst ? "LSBFirst" : "MSBFirst",
           img->depth);
    printf("XImage: red_mask=0x%lx green_mask=0x%lx blue_mask=0x%lx\n",
           img->red_mask, img->green_mask, img->blue_mask);
    printf("XImage: bytes_per_line=%d\n", img->bytes_per_line);

    // Check first few pixels
    printf("First 20 bytes of image data:\n");
    unsigned char *data = (unsigned char*)img->data;
    for (int i = 0; i < 20; i++) {
        printf("%02x ", data[i]);
    }
    printf("\n");

    // Check pixel at (50, 50)
    unsigned long pixel = XGetPixel(img, 50, 50);
    printf("Pixel at (50,50): 0x%08lx\n", pixel);

    // Extract RGB using XGetPixel (proper way)
    int r = (pixel & img->red_mask) >> 16;
    int g = (pixel & img->green_mask) >> 8;
    int b = (pixel & img->blue_mask);
    printf("RGB at (50,50): R=%d G=%d B=%d\n", r, g, b);

    XDestroyImage(img);
    XCloseDisplay(display);
}
*/
import "C"

func main() {
    C.debug_capture()
}
