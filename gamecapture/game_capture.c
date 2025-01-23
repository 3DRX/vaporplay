#include "game_capture.h"

void copyBGR24(void *dst, char *src, size_t l) { // 64bit aligned copy
  uint64_t *d = (uint64_t *)dst;
  uint64_t *s = (uint64_t *)src;
  l /= 8;
  for (size_t i = 0; i < l; i++) {
    uint64_t v = *s;
    // Reorder BGR to RGB
    *d = 0xFF000000FF000000 | ((v >> 16) & 0xFF00000000) |
         (v & 0xFF0000000000) | ((v & 0xFF00000000) << 16) |
         ((v >> 16) & 0xFF) | (v & 0xFF00) | ((v & 0xFF) << 16);
    d++;
    s++;
  }
}

void copyBGR16(void *dst, char *src, size_t l) { // 64bit aligned copy
  uint64_t *d = (uint64_t *)dst;
  uint32_t *s = (uint32_t *)src;
  l /= 8;
  for (size_t i = 0; i < l; i++) {
    uint64_t v = *s;
    // Reorder BGR to RGB
    *d = 0xFF000000FF000000 | ((v & 0xF8000000) << 8) |
         ((v & 0x7E00000) << 21) | ((v & 0x1F0000) << 35) |
         ((v & 0xF800) >> 8) | ((v & 0x7E0) << 5) | ((v & 0x1F) << 19);
    d++;
    s++;
  }
}

char *align64(char *ptr) { // return 64bit aligned pointer
  if (((size_t)ptr & 0x07) == 0) {
    return ptr;
  }
  // Clear lower 3bits to align the address to 8bytes.
  return (char *)(((size_t)ptr & (~(size_t)0x07)) + 0x08);
}
size_t align64ForTest(size_t ptr) { return (size_t)align64((char *)ptr); }
