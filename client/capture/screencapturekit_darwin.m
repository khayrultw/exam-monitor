//go:build darwin && cgo
// +build darwin,cgo

#import <Foundation/Foundation.h>
#import <ScreenCaptureKit/ScreenCaptureKit.h>
#import <CoreMedia/CoreMedia.h>
#import <CoreVideo/CoreVideo.h>
#import "screencapturekit_darwin.h"

@interface SCKStreamDelegate : NSObject <SCStreamDelegate, SCStreamOutput>
@property (nonatomic) CVPixelBufferRef latestFrame;
@property (nonatomic) dispatch_semaphore_t frameSemaphore;
@property (nonatomic) int width;
@property (nonatomic) int height;
@end

@implementation SCKStreamDelegate

- (instancetype)init {
    self = [super init];
    if (self) {
        _frameSemaphore = dispatch_semaphore_create(1);
        _latestFrame = NULL;
    }
    return self;
}

- (void)stream:(SCStream *)stream 
    didOutputSampleBuffer:(CMSampleBufferRef)sampleBuffer 
    ofType:(SCStreamOutputType)type {
    
    if (type != SCStreamOutputTypeScreen) return;
    
    CVPixelBufferRef pixelBuffer = CMSampleBufferGetImageBuffer(sampleBuffer);
    if (!pixelBuffer) return;
    
    dispatch_semaphore_wait(_frameSemaphore, DISPATCH_TIME_FOREVER);
    
    if (_latestFrame) {
        CVPixelBufferRelease(_latestFrame);
    }
    
    _latestFrame = CVPixelBufferRetain(pixelBuffer);
    _width = (int)CVPixelBufferGetWidth(pixelBuffer);
    _height = (int)CVPixelBufferGetHeight(pixelBuffer);
    
    dispatch_semaphore_signal(_frameSemaphore);
}

- (void)dealloc {
    if (_latestFrame) {
        CVPixelBufferRelease(_latestFrame);
    }
}

@end

typedef struct SCKCapture {
    SCStream *stream;
    SCKStreamDelegate *delegate;
    dispatch_queue_t queue;
    int width;
    int height;
    int running;
} SCKCapture;

SCKCapture* sck_capture_init(int *out_width, int *out_height, char **error) {
    if (@available(macOS 12.3, *)) {
        SCKCapture *cap = (SCKCapture*)calloc(1, sizeof(SCKCapture));
        if (!cap) {
            if (error) *error = strdup("Memory allocation failed");
            return NULL;
        }
        
        __block SCShareableContent *content = nil;
        __block NSError *err = nil;
        dispatch_semaphore_t sema = dispatch_semaphore_create(0);
        
        // Get shareable content (displays)
        [SCShareableContent getShareableContentWithCompletionHandler:^(SCShareableContent *shareableContent, NSError *error) {
            content = shareableContent;
            err = error;
            dispatch_semaphore_signal(sema);
        }];
        
        dispatch_semaphore_wait(sema, DISPATCH_TIME_FOREVER);
        
        if (err || !content || content.displays.count == 0) {
            if (error) {
                NSString *msg = err ? err.localizedDescription : @"No displays found";
                *error = strdup([msg UTF8String]);
            }
            free(cap);
            return NULL;
        }
        
        // Get main display
        SCDisplay *display = content.displays.firstObject;
        cap->width = (int)display.width;
        cap->height = (int)display.height;
        *out_width = cap->width;
        *out_height = cap->height;
        
        // Configure stream
        SCStreamConfiguration *config = [[SCStreamConfiguration alloc] init];
        config.width = display.width;
        config.height = display.height;
        config.minimumFrameInterval = CMTimeMake(1, 8); // 8 FPS
        config.pixelFormat = kCVPixelFormatType_32BGRA;
        config.showsCursor = YES;
        config.queueDepth = 3;
        
        // Create content filter (capture entire display)
        SCContentFilter *filter = [[SCContentFilter alloc] initWithDisplay:display 
                                                          excludingWindows:@[]];
        
        // Create delegate
        cap->delegate = [[SCKStreamDelegate alloc] init];
        
        // Create stream
        cap->stream = [[SCStream alloc] initWithFilter:filter 
                                         configuration:config 
                                              delegate:cap->delegate];
        
        if (!cap->stream) {
            if (error) *error = strdup("Failed to create stream");
            free(cap);
            return NULL;
        }
        
        // Create queue for callbacks
        cap->queue = dispatch_queue_create("com.examguard.screencapture", 
                                          DISPATCH_QUEUE_SERIAL);
        
        // Add stream output
        err = nil;
        [cap->stream addStreamOutput:cap->delegate 
                                type:SCStreamOutputTypeScreen 
                  sampleHandlerQueue:cap->queue 
                               error:&err];
        
        if (err) {
            if (error) *error = strdup([[err localizedDescription] UTF8String]);
            free(cap);
            return NULL;
        }
        
        return cap;
    } else {
        if (error) *error = strdup("ScreenCaptureKit requires macOS 12.3 or later");
        return NULL;
    }
}

int sck_capture_start(SCKCapture *cap) {
    if (!cap || !cap->stream) return 0;
    
    if (@available(macOS 12.3, *)) {
        __block BOOL success = NO;
        __block NSError *err = nil;
        dispatch_semaphore_t sema = dispatch_semaphore_create(0);
        
        [cap->stream startCaptureWithCompletionHandler:^(NSError *error) {
            success = (error == nil);
            err = error;
            dispatch_semaphore_signal(sema);
        }];
        
        dispatch_semaphore_wait(sema, DISPATCH_TIME_FOREVER);
        
        if (success) {
            cap->running = 1;
            return 1;
        }
        return 0;
    }
    return 0;
}

int sck_capture_get_frame(SCKCapture *cap, uint8_t *rgba_out) {
    if (!cap || !cap->delegate || !cap->running) return -1;
    
    dispatch_semaphore_wait(cap->delegate.frameSemaphore, DISPATCH_TIME_FOREVER);
    
    if (!cap->delegate.latestFrame) {
        dispatch_semaphore_signal(cap->delegate.frameSemaphore);
        return 0; // No frame available
    }
    
    CVPixelBufferRef pixelBuffer = cap->delegate.latestFrame;
    CVPixelBufferLockBaseAddress(pixelBuffer, kCVPixelBufferLock_ReadOnly);
    
    size_t width = CVPixelBufferGetWidth(pixelBuffer);
    size_t height = CVPixelBufferGetHeight(pixelBuffer);
    size_t bytesPerRow = CVPixelBufferGetBytesPerRow(pixelBuffer);
    uint8_t *baseAddress = (uint8_t *)CVPixelBufferGetBaseAddress(pixelBuffer);
    
    // Convert BGRA to RGBA
    for (size_t y = 0; y < height; y++) {
        for (size_t x = 0; x < width; x++) {
            size_t src_idx = y * bytesPerRow + x * 4;
            size_t dst_idx = y * width * 4 + x * 4;
            
            rgba_out[dst_idx + 0] = baseAddress[src_idx + 2]; // R
            rgba_out[dst_idx + 1] = baseAddress[src_idx + 1]; // G
            rgba_out[dst_idx + 2] = baseAddress[src_idx + 0]; // B
            rgba_out[dst_idx + 3] = 255;                       // A
        }
    }
    
    CVPixelBufferUnlockBaseAddress(pixelBuffer, kCVPixelBufferLock_ReadOnly);
    
    // Clear the frame so we don't return it again
    CVPixelBufferRelease(cap->delegate.latestFrame);
    cap->delegate.latestFrame = NULL;
    
    dispatch_semaphore_signal(cap->delegate.frameSemaphore);
    
    return 1; // Frame captured successfully
}

void sck_capture_get_size(SCKCapture *cap, int *width, int *height) {
    if (cap && cap->delegate) {
        *width = cap->delegate.width > 0 ? cap->delegate.width : cap->width;
        *height = cap->delegate.height > 0 ? cap->delegate.height : cap->height;
    } else {
        *width = 0;
        *height = 0;
    }
}

void sck_capture_stop(SCKCapture *cap) {
    if (!cap || !cap->stream || !cap->running) return;
    
    if (@available(macOS 12.3, *)) {
        __block dispatch_semaphore_t sema = dispatch_semaphore_create(0);
        
        [cap->stream stopCaptureWithCompletionHandler:^(NSError *error) {
            dispatch_semaphore_signal(sema);
        }];
        
        dispatch_semaphore_wait(sema, DISPATCH_TIME_FOREVER);
        cap->running = 0;
    }
}

void sck_capture_destroy(SCKCapture *cap) {
    if (!cap) return;
    
    if (cap->running) {
        sck_capture_stop(cap);
    }
    
    if (@available(macOS 12.3, *)) {
        if (cap->stream) {
            cap->stream = nil;
        }
        if (cap->delegate) {
            cap->delegate = nil;
        }
    }
    
    free(cap);
}