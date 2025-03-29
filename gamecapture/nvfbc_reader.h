#ifndef NVFBC_READER_H
#define NVFBC_READER_H

#include "NvFBC.h"

NVFBC_API_FUNCTION_LIST nvfbc_get_functions();
NVFBC_SESSION_HANDLE nvfbc_create_session(NVFBC_API_FUNCTION_LIST pFn);
NVFBC_CREATE_CAPTURE_SESSION_PARAMS
nvfbc_create_session_params(NVFBC_API_FUNCTION_LIST pFn,
                            NVFBC_SESSION_HANDLE fbcHandle);
unsigned char *nvfbc_setup(NVFBC_API_FUNCTION_LIST pFn,
                           NVFBC_SESSION_HANDLE fbcHandle);
NVFBC_FRAME_GRAB_INFO
nvfbc_create_grab_frame_info(NVFBC_API_FUNCTION_LIST pFn,
                             NVFBC_SESSION_HANDLE fbcHandle);
NVFBC_TOSYS_GRAB_FRAME_PARAMS
nvfbc_create_grab_frame_params(NVFBC_API_FUNCTION_LIST pFn,
                               NVFBC_SESSION_HANDLE fbcHandle,
                               NVFBC_FRAME_GRAB_INFO frameInfo);
void nvfbc_grab_frame(NVFBC_API_FUNCTION_LIST pFn,
                      NVFBC_SESSION_HANDLE fbcHandle,
                      NVFBC_TOSYS_GRAB_FRAME_PARAMS grabParams);
void nvfbc_destroy_session(NVFBC_API_FUNCTION_LIST pFn,
                           NVFBC_SESSION_HANDLE fbcHandle);

#endif
