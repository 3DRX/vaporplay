package ffmpeg

import (
	"github.com/pion/mediadevices/pkg/codec"
)

type Params struct {
	codec.BaseParams
	codecName string
}
