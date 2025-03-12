package gamecapture

/*
#cgo LDFLAGS: -lX11 -lXext
#include "game_capture.h"
#include "window_match.h"
#include <X11/Xlib.h>
#include <stdint.h>
#include <string.h>
#include <sys/shm.h>
#define XUTIL_DEFINE_FUNCTIONS
#include <X11/Xutil.h>
#include <X11/extensions/XShm.h>
#include <stdlib.h>
#include <stdio.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"unsafe"
)

const shmaddrInvalid = ^uintptr(0)

type windowmatch C.WindowMatch

type pixelFormat int

const (
	pixFmtBGR24 pixelFormat = iota
	pixFmtRGB24
	pixFmtBGR16
	pixFmtRGB16
)

func openWindow(windowname string) (*windowmatch, error) {
	cstr := C.CString(windowname)
	defer C.free(unsafe.Pointer(cstr))
	wm := C.query_window_by_name(cstr)
	if wm == nil {
		return nil, errors.New("failed to open display")
	}
	return (*windowmatch)(wm), nil
}

func (wm *windowmatch) Close() {
	C.XCloseDisplay(wm.display)
	C.free(unsafe.Pointer(wm))
}

type shmImage struct {
	dp     *C.Display
	img    *C.XImage
	shm    C.XShmSegmentInfo
	b      []byte
	pixFmt pixelFormat
}

func (s *shmImage) Free() {
	if s.img != nil {
		C.shmdt(unsafe.Pointer(s.shm.shmaddr))
		C.XShmDetach(s.dp, &s.shm)
		C.XDestroyImage(s.img)
	}
}

func (s *shmImage) ColorModel() color.Model {
	return color.RGBAModel
}

func (s *shmImage) Bounds() image.Rectangle {
	return image.Rect(0, 0, int(s.img.width), int(s.img.height))
}

type colorFunc func() (r, g, b, a uint32)

func (c colorFunc) RGBA() (r, g, b, a uint32) {
	return c()
}

func (s *shmImage) At(x, y int) color.Color {
	switch s.pixFmt {
	case pixFmtRGB24:
		addr := (x + y*int(s.img.width)) * 4
		r := uint32(s.b[addr]) * 0x100
		g := uint32(s.b[addr+1]) * 0x100
		b := uint32(s.b[addr+2]) * 0x100
		return colorFunc(func() (_, _, _, _ uint32) {
			return r, g, b, 0xFFFF
		})
	case pixFmtBGR24:
		addr := (x + y*int(s.img.width)) * 4
		b := uint32(s.b[addr]) * 0x100
		g := uint32(s.b[addr+1]) * 0x100
		r := uint32(s.b[addr+2]) * 0x100
		return colorFunc(func() (_, _, _, _ uint32) {
			return r, g, b, 0xFFFF
		})
	case pixFmtRGB16:
		addr := (x + y*int(s.img.width)) * 2
		b1, b2 := s.b[addr], s.b[addr+1]
		r := uint32(b1>>3) * 0x100
		g := uint32((b1&0x7)<<3|(b2&0xE0)>>5) * 0x100
		b := uint32(b2&0x1F) * 0x100
		return colorFunc(func() (_, _, _, _ uint32) {
			return r, g, b, 0xFFFF
		})
	case pixFmtBGR16:
		addr := (x + y*int(s.img.width)) * 2
		b1, b2 := s.b[addr], s.b[addr+1]
		b := uint32(b1>>3) * 0x100
		g := uint32((b1&0x7)<<3|(b2&0xE0)>>5) * 0x100
		r := uint32(b2&0x1F) * 0x100
		return colorFunc(func() (_, _, _, _ uint32) {
			return r, g, b, 0xFFFF
		})
	default:
		panic("unsupported pixel format")
	}
}

func (s *shmImage) RGBAAt(x, y int) color.RGBA {
	switch s.pixFmt {
	case pixFmtRGB24:
		addr := (x + y*int(s.img.width)) * 4
		r := s.b[addr]
		g := s.b[addr+1]
		b := s.b[addr+2]
		return color.RGBA{R: r, G: g, B: b, A: 0xFF}
	case pixFmtBGR24:
		addr := (x + y*int(s.img.width)) * 4
		b := s.b[addr]
		g := s.b[addr+1]
		r := s.b[addr+2]
		return color.RGBA{R: r, G: g, B: b, A: 0xFF}
	case pixFmtRGB16:
		addr := (x + y*int(s.img.width)) * 2
		b1, b2 := s.b[addr], s.b[addr+1]
		r := b1 >> 3
		g := (b1&0x7)<<3 | (b2&0xE0)>>5
		b := b2 & 0x1F
		return color.RGBA{R: r, G: g, B: b, A: 0xFF}
	case pixFmtBGR16:
		addr := (x + y*int(s.img.width)) * 2
		b1, b2 := s.b[addr], s.b[addr+1]
		b := b1 >> 3
		g := (b1&0x7)<<3 | (b2&0xE0)>>5
		r := b2 & 0x1F
		return color.RGBA{R: r, G: g, B: b, A: 0xFF}
	default:
		panic("unsupported pixel format")
	}
}

// ToRGBA actually convert image as BGRA format, but in RGBA struct.
// Later in nvenc, we will use BGRA format,
// so we can reduce memory copy when the X11 piexl format is BGR (which is for most cases).
func (s *shmImage) ToRGBA(dst *image.RGBA) *image.RGBA {
	dst.Rect = s.Bounds()
	dst.Stride = int(s.img.width) * 4
	l := int(4 * s.img.width * s.img.height)
	if len(dst.Pix) < l {
		if cap(dst.Pix) < l {
			dst.Pix = make([]uint8, l)
		}
		dst.Pix = dst.Pix[:l]
	}
	switch s.pixFmt {
	case pixFmtRGB24:
		// C.memcpy(unsafe.Pointer(&dst.Pix[0]), unsafe.Pointer(s.img.data), C.size_t(len(dst.Pix)))
		// Since we use BGRA pixel format later in nvenc, we need to turn rgb to bgr
		C.copyBGR24(unsafe.Pointer(&dst.Pix[0]), s.img.data, C.size_t(len(dst.Pix)))
		return dst
	case pixFmtBGR24:
		// C.copyBGR24(unsafe.Pointer(&dst.Pix[0]), s.img.data, C.size_t(len(dst.Pix)))
		// try a creazy hack, since nvenc supports BGRA, we just package BGRA as RGBA,
		// and select format BGRA in libavcodec.
		// By doing this, hopefully we can reduce memory copy and improve performance.
		C.memcpy(unsafe.Pointer(&dst.Pix[0]), unsafe.Pointer(s.img.data), C.size_t(len(dst.Pix)))
		return dst
	case pixFmtRGB16:
		// C.memcpy(unsafe.Pointer(&dst.Pix[0]), unsafe.Pointer(s.img.data), C.size_t(len(dst.Pix)))
		C.copyBGR16(unsafe.Pointer(&dst.Pix[0]), s.img.data, C.size_t(len(dst.Pix)))
		return dst
	case pixFmtBGR16:
		// C.copyBGR16(unsafe.Pointer(&dst.Pix[0]), s.img.data, C.size_t(len(dst.Pix)))
		C.memcpy(unsafe.Pointer(&dst.Pix[0]), unsafe.Pointer(s.img.data), C.size_t(len(dst.Pix)))
		return dst
	default:
		panic("unsupported pixel format")
	}
}

func newShmImage(dp *C.Display, window C.Window) (*shmImage, error) {
	windAttrs := C.XWindowAttributes{}
	if res := C.XGetWindowAttributes(dp, window, &windAttrs); res == 0 {
		return nil, errors.New("failed to get window attributes")
	}

	fmt.Printf("Capturing window %dx%d ...\n", windAttrs.width, windAttrs.height)
	w := int(windAttrs.width)
	h := int(windAttrs.height)
	v := windAttrs.visual
	depth := int(windAttrs.depth)

	s := &shmImage{dp: dp}

	switch {
	case v.red_mask == 0xFF && v.green_mask == 0xFF00 && v.blue_mask == 0xFF0000:
		s.pixFmt = pixFmtRGB24
	case v.red_mask == 0xFF0000 && v.green_mask == 0xFF00 && v.blue_mask == 0xFF:
		s.pixFmt = pixFmtBGR24
	case v.red_mask == 0x1F && v.green_mask == 0x7E0 && v.blue_mask == 0xF800:
		s.pixFmt = pixFmtRGB16
	case v.red_mask == 0xF800 && v.green_mask == 0x7E0 && v.blue_mask == 0x1F:
		s.pixFmt = pixFmtBGR16
	default:
		fmt.Printf("x11capture: unsupported pixel format (R: %0x, G: %0x, B: %0x)\n",
			v.red_mask, v.green_mask, v.blue_mask)
		return nil, errors.New("unsupported pixel format")
	}

	s.shm.shmid = C.shmget(C.IPC_PRIVATE, C.size_t(w*h*4+8), C.IPC_CREAT|0600)
	if s.shm.shmid == -1 {
		return nil, errors.New("failed to get shared memory")
	}
	s.shm.shmaddr = (*C.char)(C.shmat(s.shm.shmid, unsafe.Pointer(nil), 0))
	if uintptr(unsafe.Pointer(s.shm.shmaddr)) == shmaddrInvalid {
		s.shm.shmaddr = nil
		return nil, errors.New("failed to get shared memory address")
	}
	s.shm.readOnly = 0
	C.shmctl(s.shm.shmid, C.IPC_RMID, nil)

	s.img = C.XShmCreateImage(
		dp, v, C.uint(depth), C.ZPixmap, C.align64(s.shm.shmaddr), &s.shm, C.uint(w), C.uint(h))
	if s.img == nil {
		s.Free()
		return nil, errors.New("failed to create XShm image")
	}
	C.XShmAttach(dp, &s.shm)
	C.XSync(dp, 0)

	return s, nil
}

type reader struct {
	img *shmImage
	wm  *windowmatch
}

func getShmImageFromWindowMatch(wm *windowmatch) (*shmImage, error) {
	if C.XShmQueryExtension(wm.display) == 0 {
		return nil, errors.New("no XShm support")
	}

	img, err := newShmImage(wm.display, wm.window)
	if err != nil {
		wm.Close()
		return nil, err
	}

	return img, nil
}

func newReader(windowname string) (*reader, error) {
	wm, err := openWindow(windowname)
	if err != nil || wm == nil {
		return nil, errors.New("failed to open display")
	}

	img, err := getShmImageFromWindowMatch(wm)
	if err != nil {
		return nil, err
	}

	return &reader{
		img: img,
		wm:  wm,
	}, nil
}

func (r *reader) Size() (int, int) {
	return int(r.img.img.width), int(r.img.img.height)
}

func (r *reader) Read() *shmImage {
	C.XShmGetImage(r.wm.display, r.wm.window, r.img.img, 0, 0, C.AllPlanes)
	r.img.b = C.GoBytes(
		unsafe.Pointer(r.img.img.data),
		C.int(r.img.img.width*r.img.img.height*4),
	)
	return r.img
}

func (r *reader) Close() {
	r.img.Free()
	r.wm.Close()
}
