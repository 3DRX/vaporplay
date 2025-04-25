package ui

import (
	"image"
	"math/rand"
)

func VideoRecord() func() image.Image {
	colors := [][3]byte{
		{235, 128, 128},
		{210, 16, 146},
		{170, 166, 16},
		{145, 54, 34},
		{107, 202, 222},
		{82, 90, 240},
		{41, 240, 110},
	}

	w := 1280
	h := 720

	yi := w * h
	ci := yi / 2
	yy := make([]byte, yi)
	cb := make([]byte, ci)
	cr := make([]byte, ci)
	yyBase := make([]byte, yi)
	cbBase := make([]byte, ci)
	crBase := make([]byte, ci)
	hColorBarEnd := h * 3 / 4
	wGradationEnd := w * 5 / 7
	for y := range hColorBarEnd {
		yi := w * y
		ci := w * y / 2
		// Color bar
		for x := range w {
			c := x * 7 / w
			yyBase[yi+x] = uint8(uint16(colors[c][0]) * 75 / 100)
			cbBase[ci+x/2] = colors[c][1]
			crBase[ci+x/2] = colors[c][2]
		}
	}
	for y := hColorBarEnd; y < h; y++ {
		yi := w * y
		ci := w * y / 2
		for x := range wGradationEnd {
			// Gray gradation
			yyBase[yi+x] = uint8(x * 255 / wGradationEnd)
			cbBase[ci+x/2] = 128
			crBase[ci+x/2] = 128
		}
		for x := wGradationEnd; x < w; x++ {
			// Noise area
			cbBase[ci+x/2] = 128
			crBase[ci+x/2] = 128
		}
	}
	random := rand.New(rand.NewSource(0))

	return func() image.Image {
		copy(yy, yyBase)
		copy(cb, cbBase)
		copy(cr, crBase)
		for y := hColorBarEnd; y < h; y++ {
			yi := w * y
			for x := wGradationEnd; x < w; x++ {
				// Noise
				yy[yi+x] = uint8(random.Int31n(2) * 255)
			}
		}
		return &image.YCbCr{
			Y:              yy,
			YStride:        w,
			Cb:             cb,
			Cr:             cr,
			CStride:        w / 2,
			SubsampleRatio: image.YCbCrSubsampleRatio422,
			Rect:           image.Rect(0, 0, w, h),
		}
	}

}
