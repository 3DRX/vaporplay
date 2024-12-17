#ifndef WINDOW_MATCH_H
#define WINDOW_MATCH_H

#include <X11/Xlib.h>
#include <X11/Xutil.h>

typedef struct {
  Display *display;
  Window window;
} WindowMatch;

WindowMatch *query_window_by_name(const char *window_name);

#endif // !WINDOW_MATCH_H
