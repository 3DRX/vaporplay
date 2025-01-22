#include "window_match.h"
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <X11/extensions/XShm.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/ipc.h>
#include <sys/shm.h>
#include <sys/stat.h>
#include <unistd.h>
#include <vpx/vp8cx.h>
#include <vpx/vpx_encoder.h>
#include <vpx/vpx_image.h>

#define STB_IMAGE_WRITE_IMPLEMENTATION
#include "stb_image_write.h"

typedef struct {
  char *output_dir;
  int frame_count;
} FrameWriter;

typedef struct {
  XImage *image;
  XShmSegmentInfo shminfo;
} SharedImage;

SharedImage *create_shared_image(Display *display, Window window) {
  SharedImage *shared = malloc(sizeof(SharedImage));
  if (!shared)
    return NULL;

  XWindowAttributes attrs;
  if (!XGetWindowAttributes(display, window, &attrs)) {
    free(shared);
    return NULL;
  }

  // Create shared memory XImage
  shared->image =
      XShmCreateImage(display, attrs.visual, attrs.depth, ZPixmap, NULL,
                      &shared->shminfo, attrs.width, attrs.height);

  if (!shared->image) {
    free(shared);
    return NULL;
  }

  // Allocate shared memory
  shared->shminfo.shmid =
      shmget(IPC_PRIVATE, shared->image->bytes_per_line * shared->image->height,
             IPC_CREAT | 0777);
  if (shared->shminfo.shmid == -1) {
    XDestroyImage(shared->image);
    free(shared);
    return NULL;
  }

  // Attach shared memory
  shared->shminfo.shmaddr = (char *)shmat(shared->shminfo.shmid, 0, 0);
  if (shared->shminfo.shmaddr == (char *)-1) {
    shmctl(shared->shminfo.shmid, IPC_RMID, 0);
    XDestroyImage(shared->image);
    free(shared);
    return NULL;
  }

  shared->image->data = shared->shminfo.shmaddr;
  shared->shminfo.readOnly = False;

  // Attach the shared memory to the X server
  if (!XShmAttach(display, &shared->shminfo)) {
    shmdt(shared->shminfo.shmaddr);
    shmctl(shared->shminfo.shmid, IPC_RMID, 0);
    XDestroyImage(shared->image);
    free(shared);
    return NULL;
  }

  return shared;
}

FrameWriter *create_frame_writer(const char *output_dir) {
  FrameWriter *writer = malloc(sizeof(FrameWriter));
  if (!writer)
    return NULL;

  writer->output_dir = strdup(output_dir);
  writer->frame_count = 0;

  return writer;
}

int save_frame(FrameWriter *writer, XImage *image) {
  char filename[256];
  snprintf(filename, sizeof(filename), "%s/frame_%06d.png", writer->output_dir,
           writer->frame_count++);

  // Convert XImage data to RGB
  unsigned char *rgb_data = malloc(image->width * image->height * 3);
  if (!rgb_data)
    return 0;

  for (int y = 0; y < image->height; y++) {
    for (int x = 0; x < image->width; x++) {
      unsigned long pixel = XGetPixel(image, x, y);
      int idx = (y * image->width + x) * 3;
      rgb_data[idx] = (pixel >> 16) & 0xFF;    // R
      rgb_data[idx + 1] = (pixel >> 8) & 0xFF; // G
      rgb_data[idx + 2] = pixel & 0xFF;        // B
    }
  }

  int success = stbi_write_png(filename, image->width, image->height, 3,
                               rgb_data, image->width * 3);

  free(rgb_data);
  return success;
}

void destroy_frame_writer(FrameWriter *writer) {
  if (writer) {
    free(writer->output_dir);
    free(writer);
  }
}

void destroy_shared_image(Display *display, SharedImage *shared) {
  if (!shared)
    return;

  XShmDetach(display, &shared->shminfo);
  XDestroyImage(shared->image);
  shmdt(shared->shminfo.shmaddr);
  shmctl(shared->shminfo.shmid, IPC_RMID, 0);
  free(shared);
}

// Function to capture a single frame
int capture_frame(Display *display, Window window, SharedImage *shared) {
  XWindowAttributes attrs;
  if (!XGetWindowAttributes(display, window, &attrs)) {
    return 0;
  }

  // Capture the window content
  return XShmGetImage(display, window, shared->image, 0, 0, AllPlanes);
}

void capture_and_encode_window(Display *display, Window window,
                               int duration_seconds, int fps) {
  XWindowAttributes attrs;
  if (!XGetWindowAttributes(display, window, &attrs)) {
    return;
  }

  printf("Capturing window %dx%d ...\n", attrs.width, attrs.height);

  SharedImage *shared = create_shared_image(display, window);
  if (!shared) {
    fprintf(stderr, "Failed to create shared image\n");
    return;
  }

  // Create frame writer instead of VPX encoder
  FrameWriter *writer = create_frame_writer("frames");
  if (!writer) {
    destroy_shared_image(display, shared);
    fprintf(stderr, "Failed to initialize frame writer\n");
    return;
  }

  printf("Recording %d seconds at %d fps...\n", duration_seconds, fps);

  int total_frames = duration_seconds * fps;
  printf("Total frames: %d\n", total_frames);
  int frame_delay_us = 1000000 / fps;

  // Create output directory if it doesn't exist
  mkdir("frames", 0777);

  for (int i = 0; i < total_frames; i++) {
    if (XShmGetImage(display, window, shared->image, 0, 0, AllPlanes)) {
      save_frame(writer, shared->image);
    }
    usleep(frame_delay_us);
  }

  printf("Finished recording\n");
  destroy_frame_writer(writer);
  destroy_shared_image(display, shared);
}

// int main(int argc, char **argv) {
//   if (argc != 2) {
//     printf("Usage: %s <window_name>\n", argv[0]);
//     return 1;
//   }
//   const char *search_name = argv[1];
//   WindowMatch *match = query_window_by_name(search_name);

//   if (!match) {
//     printf("No window found containing: %s\n", search_name);
//     return 1;
//   }

//   printf("Found window with ID: 0x%lx\n", match->window);
//   // Example: Get window name to verify
//   char *window_name = NULL;
//   if (XFetchName(match->display, match->window, &window_name)) {
//     if (window_name) {
//       printf("Window name: %s\n", window_name);
//       XFree(window_name);
//     }
//   }

//   capture_and_encode_window(match->display, match->window, 5, 60);

//   // Clean up
//   XCloseDisplay(match->display);
//   free(match);
//   return 0;
// }
