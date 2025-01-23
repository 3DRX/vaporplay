#ifndef GAME_CAPTURE_H
#define GAME_CAPTURE_H

#include <X11/Xlib.h>
#include <stdint.h>
#include <string.h>
#include <sys/shm.h>
#define XUTIL_DEFINE_FUNCTIONS
#include <X11/Xutil.h>
#include <X11/extensions/XShm.h>
#include "window_match.h"

void copyBGR24(void *dst, char *src, size_t l); // 64bit aligned copy

void copyBGR16(void *dst, char *src, size_t l); // 64bit aligned copy

char *align64(char *ptr); // return 64bit aligned pointer

size_t align64ForTest(size_t ptr);

#endif
