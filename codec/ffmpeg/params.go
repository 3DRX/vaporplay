package ffmpeg

import (
	"github.com/asticode/go-astiav"
	"github.com/pion/mediadevices/pkg/codec"
)

type Params struct {
	codec.BaseParams
	codecName      string
	hardwareDevice string
	pixelFormat    astiav.PixelFormat
	FrameRate      float32
}
