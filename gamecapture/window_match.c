#include "window_match.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

void find_window_by_name_recursive(Display *display, Window window,
                                   const char *search_name,
                                   WindowMatch *result) {
  Window root, parent, *children;
  unsigned int nchildren;
  XWindowAttributes attrs;
  char *window_name = NULL;
  XClassHint class_hint;
  XTextProperty text_prop;

  // If we already found a match, return early
  if (result->window != None) {
    return;
  }

  // Get window attributes
  if (!XGetWindowAttributes(display, window, &attrs)) {
    return;
  }

  // Check window name from XFetchName
  if (XFetchName(display, window, &window_name)) {
    if (window_name && strstr(window_name, search_name)) {
      result->window = window;
      XFree(window_name);
      return;
    }
    if (window_name)
      XFree(window_name);
  }

  // Check WM_NAME property
  if (XGetWMName(display, window, &text_prop)) {
    if (text_prop.value && strstr((char *)text_prop.value, search_name)) {
      result->window = window;
      XFree(text_prop.value);
      return;
    }
    if (text_prop.value)
      XFree(text_prop.value);
  }

  // Check window class
  if (XGetClassHint(display, window, &class_hint)) {
    if ((class_hint.res_name && strstr(class_hint.res_name, search_name)) ||
        (class_hint.res_class && strstr(class_hint.res_class, search_name))) {
      result->window = window;
      XFree(class_hint.res_name);
      XFree(class_hint.res_class);
      return;
    }
    XFree(class_hint.res_name);
    XFree(class_hint.res_class);
  }

  // Recursively search children
  if (XQueryTree(display, window, &root, &parent, &children, &nchildren)) {
    for (unsigned int i = 0; i < nchildren; i++) {
      find_window_by_name_recursive(display, children[i], search_name, result);
      if (result->window != None) {
        XFree(children);
        return;
      }
    }
    if (children)
      XFree(children);
  }
}

WindowMatch *query_window_by_name(const char *window_name) {
  WindowMatch *result = malloc(sizeof(WindowMatch));
  if (!result) {
    printf("windowmatch malloc failed\n");
    return NULL;
  }

  // Initialize result
  result->display = XOpenDisplay(getenv("DISPLAY"));
  result->window = None;

  if (!result->display) {
    free(result);
    printf("XOpenDisplay failed\n");
    return NULL;
  }

  // Get root window and start recursive search
  Window root = DefaultRootWindow(result->display);
  find_window_by_name_recursive(result->display, root, window_name, result);

  // If no window found, clean up and return NULL
  if (result->window == None) {
    XCloseDisplay(result->display);
    free(result);
    return NULL;
  }

  return result;
}
