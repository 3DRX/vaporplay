package ffmpeg

import (
	"fmt"
	"image"
	"io"
	"sync"

	"github.com/asticode/go-astiav"
	"github.com/pion/mediadevices/pkg/codec"
	"github.com/pion/mediadevices/pkg/io/video"
	"github.com/pion/mediadevices/pkg/prop"
)

type encoder struct {
	codec    *astiav.Codec
	codecCtx *astiav.CodecContext
	frame    *astiav.Frame
	packet   *astiav.Packet
	width    int
	height   int
	r        video.Reader

	mu     sync.Mutex
	closed bool
}

type VP8Params struct {
	Params
}

func NewVP8Params() (VP8Params, error) {
	return VP8Params{
		Params: Params{
			codecName: "vp8_vaapi",
		},
	}, nil
}

// RTPCodec represents the codec metadata
func (p *VP8Params) RTPCodec() *codec.RTPCodec {
	return codec.NewRTPVP8Codec(90000)
}

// BuildVideoEncoder builds VP8 encoder with given params
func (p *VP8Params) BuildVideoEncoder(r video.Reader, property prop.Media) (codec.ReadCloser, error) {
	return newEncoder(r, property, p.Params)
}

func newEncoder(r video.Reader, p prop.Media, params Params) (*encoder, error) {
	codec := astiav.FindEncoderByName(params.codecName)
	if codec == nil {
		return nil, fmt.Errorf("codec not found: %s", params.codecName)
	}

	codecCtx := astiav.AllocCodecContext(codec)
	if codecCtx == nil {
		return nil, fmt.Errorf("failed to allocate codec context")
	}

	// Configure codec context
	codecCtx.SetWidth(p.Width)
	codecCtx.SetHeight(p.Height)
	codecCtx.SetTimeBase(astiav.NewRational(1, 1000))
	codecCtx.SetFramerate(astiav.NewRational(int(p.FrameRate), 1))
	codecCtx.SetPixelFormat(astiav.PixelFormat(astiav.PixelFormatYuv420P))
	codecCtx.SetBitRate(int64(params.BitRate))
	codecCtx.SetGopSize(params.KeyFrameInterval)
	codecCtx.SetMaxBFrames(1)

	// Open codec context
	if err := codecCtx.Open(codec, nil); err != nil {
		codecCtx.Free()
		return nil, fmt.Errorf("failed to open codec context: %w", err)
	}

	frame := astiav.AllocFrame()
	if frame == nil {
		codecCtx.Free()
		return nil, fmt.Errorf("failed to allocate frame")
	}

	frame.SetWidth(p.Width)
	frame.SetHeight(p.Height)
	frame.SetPixelFormat(astiav.PixelFormat(astiav.PixelFormatYuv420P))

	// Allocate frame buffers
	// Allocate frame buffers with 32-byte alignment
	// 32 is commonly used as it's compatible with most SIMD instructions and hardware requirements
	const alignment = 32
	if err := frame.AllocBuffer(alignment); err != nil {
		frame.Free()
		codecCtx.Free()
		return nil, fmt.Errorf("failed to allocate frame buffer: %w", err)
	}

	packet := astiav.AllocPacket()
	if packet == nil {
		frame.Free()
		codecCtx.Free()
		return nil, fmt.Errorf("failed to allocate packet")
	}

	return &encoder{
		codec:    codec,
		codecCtx: codecCtx,
		frame:    frame,
		packet:   packet,
		width:    p.Width,
		height:   p.Height,
		r:        r,
	}, nil
}

func (e *encoder) Controller() codec.EncoderController {
	return e
}

func (e *encoder) Read() ([]byte, func(), error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return nil, func() {}, io.EOF
	}

	rawImg, release, err := e.r.Read()
	if err != nil {
		return nil, func() {}, err
	}
	defer release()
	img := rawImg.(*image.YCbCr)

	// Copy YCbCr data to frame
	err = e.frame.Data().FromImage(img)
	if err != nil {
		return nil, func() {}, fmt.Errorf("failed to copy image data: %w", err)
	}

	// Send frame to encoder
	if err := e.codecCtx.SendFrame(e.frame); err != nil {
		return nil, func() {}, fmt.Errorf("failed to send frame: %w", err)
	}

	// Receive encoded packet
	if err := e.codecCtx.ReceivePacket(e.packet); err != nil {
		return nil, func() {}, fmt.Errorf("failed to receive packet: %w", err)
	}

	// Copy packet data
	data := make([]byte, e.packet.Size())
	copy(data, e.packet.Data())
	e.packet.Unref()

	return data, func() {}, nil
}

// ForceKeyFrame forces the next frame to be encoded as a keyframe
func (e *encoder) ForceKeyFrame() error {
	// e.mu.Lock()
	// defer e.mu.Unlock()

	// // Set frame properties to force a keyframe
	// if err := e.frame.SetFlags(e.frame.Flags() | astiav.FrameFlagKey); err != nil {
	//     return fmt.Errorf("failed to set keyframe flag: %w", err)
	// }

	// // Set frame picture type to I-frame
	// if err := e.frame.SetPictType(astiav.PictTypeI); err != nil {
	//     return fmt.Errorf("failed to set picture type: %w", err)
	// }

	return nil
}

// SetBitrate updates the encoder's bitrate
func (e *encoder) SetBitrate(bitrate int64) error {
	// e.mu.Lock()
	// defer e.mu.Unlock()

	// // Flush the encoder
	// if err := e.codecCtx.SendFrame(nil); err != nil {
	//     return fmt.Errorf("failed to flush encoder: %w", err)
	// }

	// for {
	//     err := e.codecCtx.ReceivePacket(e.packet)
	//     if err != nil {
	//         break
	//     }
	//     e.packet.Unref()
	// }

	// // Set new bitrate
	// e.codecCtx.SetBitRate(bitrate)

	// // Some codecs might require additional parameters to be updated
	// if e.codec.Name() == "libx264" || e.codec.Name() == "h264" {
	//     // Update rate control buffer size and max rate for x264
	//     e.codecCtx.SetRCBufferSize(int(bitrate * 2))
	//     e.codecCtx.SetRCMaxRate(bitrate)
	// }

	return nil
}

func (e *encoder) Close() error {
	if e.packet != nil {
		e.packet.Free()
	}
	if e.frame != nil {
		e.frame.Free()
	}
	if e.codecCtx != nil {
		e.codecCtx.Free()
	}
	return nil
}
