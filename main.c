#include "window_match.h"
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <X11/extensions/XShm.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/ipc.h>
#include <sys/shm.h>
#include <unistd.h>
#include <vpx/vp8cx.h>
#include <vpx/vpx_encoder.h>
#include <vpx/vpx_image.h>

typedef struct {
  vpx_codec_ctx_t codec;
  vpx_codec_enc_cfg_t cfg;
  vpx_image_t *image;
  FILE *output_file;
  int frame_count;
} VpxEncoder;

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

VpxEncoder *initialize_vpx_encoder(int width, int height, int fps) {
  VpxEncoder *encoder = malloc(sizeof(VpxEncoder));
  if (!encoder)
    return NULL;

  encoder->frame_count = 0;
  encoder->output_file = fopen("output.webm", "wb");
  if (!encoder->output_file) {
    free(encoder);
    return NULL;
  }

  // Initialize codec configuration
  vpx_codec_err_t res =
      vpx_codec_enc_config_default(vpx_codec_vp8_cx(), &encoder->cfg, 0);
  if (res) {
    fclose(encoder->output_file);
    free(encoder);
    return NULL;
  }

  // Set configuration parameters
  encoder->cfg.g_w = width;
  encoder->cfg.g_h = height;
  encoder->cfg.rc_target_bitrate = 8000; // Target bitrate in kbit/s
  encoder->cfg.g_timebase.num = 1;
  encoder->cfg.g_timebase.den = fps;
  encoder->cfg.rc_end_usage = VPX_CBR;
  encoder->cfg.g_threads = 4; // Use multiple threads for encoding

  // Initialize codec
  if (vpx_codec_enc_init(&encoder->codec, vpx_codec_vp8_cx(), &encoder->cfg,
                         0)) {
    fclose(encoder->output_file);
    free(encoder);
    return NULL;
  }

  // Allocate image
  encoder->image = vpx_img_alloc(NULL, VPX_IMG_FMT_I420, width, height, 1);
  if (!encoder->image) {
    vpx_codec_destroy(&encoder->codec);
    fclose(encoder->output_file);
    free(encoder);
    return NULL;
  }

  return encoder;
}

void destroy_vpx_encoder(VpxEncoder *encoder) {
  if (!encoder)
    return;

  // Get and write any remaining packets
  const vpx_codec_cx_pkt_t *pkt;
  vpx_codec_iter_t iter = NULL;
  while ((pkt = vpx_codec_get_cx_data(&encoder->codec, &iter))) {
    if (pkt->kind == VPX_CODEC_CX_FRAME_PKT) {
      fwrite(pkt->data.frame.buf, 1, pkt->data.frame.sz, encoder->output_file);
    }
  }

  vpx_img_free(encoder->image);
  vpx_codec_destroy(&encoder->codec);
  fclose(encoder->output_file);
  free(encoder);
}

void convert_ximage_to_vpx(XImage *ximage, vpx_image_t *vpx_img) {
  // Ensure the vpx_image_t is initialized correctly
  if (vpx_img == NULL || ximage == NULL) {
    return; // Handle error appropriately
  }

  int width = ximage->width;
  int height = ximage->height;

  // Initialize vpx_image_t for I420 format
  if (vpx_img->fmt != VPX_IMG_FMT_I420 || vpx_img->d_w != width ||
      vpx_img->d_h != height) {
    // Configure vpx_image_t with proper dimensions and format
    vpx_img->fmt = VPX_IMG_FMT_I420;
    vpx_img->d_w = width;
    vpx_img->d_h = height;
    vpx_img->planes[0] = (uint8_t *)malloc(width * height); // Y plane
    vpx_img->planes[1] =
        (uint8_t *)malloc((width / 2) * (height / 2)); // U plane
    vpx_img->planes[2] =
        (uint8_t *)malloc((width / 2) * (height / 2)); // V plane
    vpx_img->stride[0] = width;
    vpx_img->stride[1] = width / 2;
    vpx_img->stride[2] = width / 2;
  }

  // Convert RGB to I420
  uint8_t *rgb_data = (uint8_t *)ximage->data;
  uint8_t *y_plane = vpx_img->planes[0];
  uint8_t *u_plane = vpx_img->planes[1];
  uint8_t *v_plane = vpx_img->planes[2];

  for (int y = 0; y < height; y++) {
    for (int x = 0; x < width; x++) {
      // Get RGB values from the XImage
      int pixel_index =
          (y * ximage->width + x) * 4; // Assuming 32 bits per pixel (ARGB)
      uint8_t r = rgb_data[pixel_index + 2]; // Red
      uint8_t g = rgb_data[pixel_index + 1]; // Green
      uint8_t b = rgb_data[pixel_index];     // Blue

      // Convert to YUV
      int Y = (0.299 * r) + (0.587 * g) + (0.114 * b);
      int U = (-0.147 * r) - (0.289 * g) + (0.436 * b) + 128;
      int V = (0.615 * r) - (0.515 * g) - (0.100 * b) + 128;

      // Clamping to [0, 255]
      Y = (Y < 0) ? 0 : (Y > 255) ? 255 : Y;
      U = (U < 0) ? 0 : (U > 255) ? 255 : U;
      V = (V < 0) ? 0 : (V > 255) ? 255 : V;

      // Fill Y plane
      y_plane[y * width + x] = (uint8_t)Y;

      // Fill U and V planes (downsampling)
      if (x % 2 == 0 && y % 2 == 0) {
        int uv_index = (y / 2) * (width / 2) + (x / 2);
        u_plane[uv_index] = (uint8_t)U;
        v_plane[uv_index] = (uint8_t)V;
      }
    }
  }
}

void encode_frame(VpxEncoder *encoder, XImage *ximage) {
  printf("Encoding frame %d\n", encoder->frame_count);
  convert_ximage_to_vpx(ximage, encoder->image);

  vpx_codec_encode(&encoder->codec, encoder->image, encoder->frame_count, 1, 0,
                   VPX_DL_REALTIME);

  const vpx_codec_cx_pkt_t *pkt;
  vpx_codec_iter_t iter = NULL;
  while ((pkt = vpx_codec_get_cx_data(&encoder->codec, &iter))) {
    if (pkt->kind == VPX_CODEC_CX_FRAME_PKT) {
      fwrite(pkt->data.frame.buf, 1, pkt->data.frame.sz, encoder->output_file);
    }
  }

  encoder->frame_count++;
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

  VpxEncoder *encoder = initialize_vpx_encoder(attrs.width, attrs.height, fps);
  if (!encoder) {
    destroy_shared_image(display, shared);
    fprintf(stderr, "Failed to initialize encoder\n");
    return;
  }

  printf("Recording %d seconds at %d fps...\n", duration_seconds, fps);

  int total_frames = duration_seconds * fps;
  printf("Total frames: %d\n", total_frames);
  int frame_delay_us = 1000000 / fps;

  for (int i = 0; i < total_frames; i++) {
    if (XShmGetImage(display, window, shared->image, 0, 0, AllPlanes)) {
      encode_frame(encoder, shared->image);
    }
    usleep(frame_delay_us);
  }

  printf("Finished recording\n");
  destroy_vpx_encoder(encoder);
  destroy_shared_image(display, shared);
}

int main(int argc, char **argv) {
  if (argc != 2) {
    printf("Usage: %s <window_name>\n", argv[0]);
    return 1;
  }
  const char *search_name = argv[1];
  WindowMatch *match = query_window_by_name(search_name);

  if (!match) {
    printf("No window found containing: %s\n", search_name);
    return 1;
  }

  printf("Found window with ID: 0x%lx\n", match->window);
  // Example: Get window name to verify
  char *window_name = NULL;
  if (XFetchName(match->display, match->window, &window_name)) {
    if (window_name) {
      printf("Window name: %s\n", window_name);
      XFree(window_name);
    }
  }

  capture_and_encode_window(match->display, match->window, 5, 60);

  // Clean up
  XCloseDisplay(match->display);
  free(match);
  return 0;
}
