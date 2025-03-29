// TODO: error handling

#include "nvfbc_reader.h"
#include "NvFBC.h"
#include <X11/Xlib.h>
#include <dlfcn.h>
#include <stdlib.h>
#include <string.h>

#define LIB_NVFBC_NAME "libnvidia-fbc.so.1"

NVFBC_API_FUNCTION_LIST nvfbc_get_functions() {
  void *libNVFBC = dlopen(LIB_NVFBC_NAME, RTLD_NOW);
  // if (libNVFBC == NULL) {
  //   return EXIT_FAILURE;
  // }

  PNVFBCCREATEINSTANCE NvFBCCreateInstance_ptr =
      (PNVFBCCREATEINSTANCE)dlsym(libNVFBC, "NvFBCCreateInstance");
  // if (NvFBCCreateInstance_ptr == NULL) {
  //   return EXIT_FAILURE;
  // }

  NVFBCSTATUS fbcStatus;
  NVFBC_API_FUNCTION_LIST pFn;
  memset(&pFn, 0, sizeof(pFn));
  pFn.dwVersion = NVFBC_VERSION;
  fbcStatus = NvFBCCreateInstance_ptr(&pFn);
  // if (fbcStatus != NVFBC_SUCCESS) {
  //   return EXIT_FAILURE;
  // }
  return pFn;
}

NVFBC_SESSION_HANDLE nvfbc_create_session(NVFBC_API_FUNCTION_LIST pFn) {
  NVFBC_SESSION_HANDLE fbcHandle;
  NVFBC_CREATE_HANDLE_PARAMS createHandleParams;
  NVFBCSTATUS fbcStatus;

  memset(&createHandleParams, 0, sizeof(createHandleParams));
  createHandleParams.dwVersion = NVFBC_CREATE_HANDLE_PARAMS_VER;

  fbcStatus = pFn.nvFBCCreateHandle(&fbcHandle, &createHandleParams);
  // if (fbcStatus != NVFBC_SUCCESS) {
  //   fprintf(stderr, "%s\n", pFn.nvFBCGetLastErrorStr(fbcHandle));
  //   return EXIT_FAILURE;
  // }

  NVFBC_GET_STATUS_PARAMS statusParams;
  memset(&statusParams, 0, sizeof(statusParams));

  statusParams.dwVersion = NVFBC_GET_STATUS_PARAMS_VER;

  fbcStatus = pFn.nvFBCGetStatus(fbcHandle, &statusParams);
  // if (fbcStatus != NVFBC_SUCCESS) {
  //   fprintf(stderr, "%s\n", pFn.nvFBCGetLastErrorStr(fbcHandle));
  //   return 1;
  // }

  // if (statusParams.bCanCreateNow == NVFBC_FALSE) {
  //   fprintf(stderr, "It is not possible to create a capture session "
  //                   "on this system.\n");
  //   return 1;
  // }

  return fbcHandle;
}

NVFBC_CREATE_CAPTURE_SESSION_PARAMS
nvfbc_create_session_params(NVFBC_API_FUNCTION_LIST pFn,
                            NVFBC_SESSION_HANDLE fbcHandle) {
  NVFBCSTATUS fbcStatus;
  unsigned int framebufferWidth, framebufferHeight;
  Display *dpy = XOpenDisplay(getenv("DISPLAY"));
  // if (dpy == NULL) {
  //   fprintf(stderr, "Unable to open display\n");
  //   return EXIT_FAILURE;
  // }

  framebufferWidth = DisplayWidth(dpy, XDefaultScreen(dpy));
  framebufferHeight = DisplayHeight(dpy, XDefaultScreen(dpy));
  NVFBC_CREATE_CAPTURE_SESSION_PARAMS createCaptureParams;
  memset(&createCaptureParams, 0, sizeof(createCaptureParams));

  createCaptureParams.dwVersion = NVFBC_CREATE_CAPTURE_SESSION_PARAMS_VER;
  createCaptureParams.eCaptureType = NVFBC_CAPTURE_TO_SYS;
  createCaptureParams.bWithCursor = NVFBC_FALSE;
  createCaptureParams.frameSize.w = framebufferWidth;
  createCaptureParams.frameSize.h = framebufferHeight;
  createCaptureParams.eTrackingType = NVFBC_TRACKING_SCREEN;
  createCaptureParams.dwSamplingRateMs = 1000 / 120;
  createCaptureParams.bAllowDirectCapture = NVFBC_TRUE;
  createCaptureParams.bPushModel = NVFBC_TRUE;
  createCaptureParams.bDisableAutoModesetRecovery = NVFBC_TRUE;

  fbcStatus = pFn.nvFBCCreateCaptureSession(fbcHandle, &createCaptureParams);
  // if (fbcStatus != NVFBC_SUCCESS) {
  //   fprintf(stderr, "%s\n", pFn.nvFBCGetLastErrorStr(fbcHandle));
  //   return 1;
  // }
  XCloseDisplay(dpy);
  return createCaptureParams;
}

unsigned char *nvfbc_setup(NVFBC_API_FUNCTION_LIST pFn,
                           NVFBC_SESSION_HANDLE fbcHandle) {
  NVFBC_TOSYS_SETUP_PARAMS setupParams;
  NVFBCSTATUS fbcStatus;
  unsigned char *frame = NULL;
  setupParams.dwVersion = NVFBC_TOSYS_SETUP_PARAMS_VER;
  setupParams.eBufferFormat = NVFBC_BUFFER_FORMAT_BGRA;
  setupParams.ppBuffer = (void **)&frame;
  setupParams.bWithDiffMap = NVFBC_FALSE;
  fbcStatus = pFn.nvFBCToSysSetUp(fbcHandle, &setupParams);
  // if (fbcStatus != NVFBC_SUCCESS) {
  //   fprintf(stderr, "%s\n", pFn.nvFBCGetLastErrorStr(fbcHandle));
  //   return EXIT_FAILURE;
  // }
  return frame;
}

NVFBC_FRAME_GRAB_INFO
nvfbc_create_grab_frame_info(NVFBC_API_FUNCTION_LIST pFn,
                             NVFBC_SESSION_HANDLE fbcHandle) {
  NVFBC_FRAME_GRAB_INFO frameInfo;
  memset(&frameInfo, 0, sizeof(frameInfo));
  return frameInfo;
}

NVFBC_TOSYS_GRAB_FRAME_PARAMS
nvfbc_create_grab_frame_params(NVFBC_API_FUNCTION_LIST pFn,
                               NVFBC_SESSION_HANDLE fbcHandle,
                               NVFBC_FRAME_GRAB_INFO frameInfo) {
  NVFBC_TOSYS_GRAB_FRAME_PARAMS grabParams;
  memset(&grabParams, 0, sizeof(grabParams));

  grabParams.dwVersion = NVFBC_TOSYS_GRAB_FRAME_PARAMS_VER;
  grabParams.dwFlags = NVFBC_TOSYS_GRAB_FLAGS_NOWAIT;
  grabParams.pFrameGrabInfo = &frameInfo;
  return grabParams;
}

void nvfbc_grab_frame(NVFBC_API_FUNCTION_LIST pFn,
                      NVFBC_SESSION_HANDLE fbcHandle,
                      NVFBC_TOSYS_GRAB_FRAME_PARAMS grabParams) {
  NVFBCSTATUS fbcStatus;
  fbcStatus = pFn.nvFBCToSysGrabFrame(fbcHandle, &grabParams);
  // if (fbcStatus != NVFBC_SUCCESS) {
  //   fprintf(stderr, "%s\n", pFn.nvFBCGetLastErrorStr(fbcHandle));
  //   return EXIT_FAILURE;
  // }
}

void nvfbc_destroy_session(NVFBC_API_FUNCTION_LIST pFn,
                           NVFBC_SESSION_HANDLE fbcHandle) {
  NVFBC_DESTROY_CAPTURE_SESSION_PARAMS destroyCaptureParams;
  NVFBC_DESTROY_HANDLE_PARAMS destroyHandleParams;
  NVFBCSTATUS fbcStatus;
  memset(&destroyCaptureParams, 0, sizeof(destroyCaptureParams));
  destroyCaptureParams.dwVersion = NVFBC_DESTROY_CAPTURE_SESSION_PARAMS_VER;
  fbcStatus = pFn.nvFBCDestroyCaptureSession(fbcHandle, &destroyCaptureParams);
  // if (fbcStatus != NVFBC_SUCCESS) {
  //   fprintf(stderr, "%s\n", pFn.nvFBCGetLastErrorStr(fbcHandle));
  //   return EXIT_FAILURE;
  // }

  memset(&destroyHandleParams, 0, sizeof(destroyHandleParams));
  destroyHandleParams.dwVersion = NVFBC_DESTROY_HANDLE_PARAMS_VER;
  fbcStatus = pFn.nvFBCDestroyHandle(fbcHandle, &destroyHandleParams);
  // if (fbcStatus != NVFBC_SUCCESS) {
  //   fprintf(stderr, "%s\n", pFn.nvFBCGetLastErrorStr(fbcHandle));
  //   return EXIT_FAILURE;
  // }
}
