package ffmpeg

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/asticode/go-astiav"
	"github.com/pion/mediadevices/pkg/codec"
	"github.com/pion/mediadevices/pkg/io/video"
	"github.com/pion/mediadevices/pkg/prop"
)

type encoder struct {
	codec       *astiav.Codec
	codecCtx    *astiav.CodecContext
	hwFramesCtx *astiav.HardwareFramesContext
	frame       *astiav.Frame
	hwFrame     *astiav.Frame
	packet      *astiav.Packet
	width       int
	height      int
	r           video.Reader

	mu     sync.Mutex
	closed bool
}

type H264Params struct {
	Params
}

func NewH264Params() (H264Params, error) {
	return H264Params{
		Params: Params{
			codecName: "h264_nvenc",
		},
	}, nil
}

// RTPCodec represents the codec metadata
func (p *H264Params) RTPCodec() *codec.RTPCodec {
	return codec.NewRTPH264Codec(90000)
}

// BuildVideoEncoder builds VP8 encoder with given params
func (p *H264Params) BuildVideoEncoder(r video.Reader, property prop.Media) (codec.ReadCloser, error) {
	readCloser, err := newEncoder(r, property, p.Params)
	if err != nil {
		slog.Error("failed to create new encoder", "error", err)
		return nil, err
	}
	slog.Info("sucsessfully created new encoder")
	return readCloser, nil
}

func newEncoder(r video.Reader, p prop.Media, params Params) (*encoder, error) {
	if p.FrameRate == 0 {
		p.FrameRate = 60
	}
	slog.Info("creating new encoder", "params", params, "props", p)
	astiav.SetLogLevel(astiav.LogLevel(astiav.LogLevelDebug))

	hwDevice, err := astiav.CreateHardwareDeviceContext(
		astiav.HardwareDeviceType(astiav.HardwareDeviceTypeCUDA),
		"/dev/dri/card1",
		nil,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

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
	codecCtx.SetTimeBase(astiav.NewRational(1, int(p.FrameRate)))
	codecCtx.SetFramerate(codecCtx.TimeBase().Invert())
	codecCtx.SetPixelFormat(astiav.PixelFormat(astiav.PixelFormatCuda))
	// codecCtx.SetBitRate(int64(params.BitRate))
	// codecCtx.SetGopSize(params.KeyFrameInterval)
	codecCtx.SetMaxBFrames(0)
	codecCtx.PrivateData().Options().Set("zerolatency", "1", 0)
	codecCtx.PrivateData().Options().Set("delay", "0", 0)

	// Create hardware frames context
	hwFramesCtx := astiav.AllocHardwareFramesContext(hwDevice)
	if hwFramesCtx == nil {
		hwDevice.Free()
		return nil, fmt.Errorf("failed to allocate hw frames context")
	}

	// Set hardware frames context parameters
	hwFramesCtx.SetWidth(p.Width)
	hwFramesCtx.SetHeight(p.Height)
	hwFramesCtx.SetHardwarePixelFormat(astiav.PixelFormat(astiav.PixelFormatCuda))
	hwFramesCtx.SetSoftwarePixelFormat(astiav.PixelFormat(astiav.PixelFormatYuv420P))
	hwFramesCtx.SetInitialPoolSize(20)

	err = hwFramesCtx.Initialize()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize hw frames context: %w", err)
	}
	codecCtx.SetHardwareFramesContext(hwFramesCtx)

	// Open codec context
	if err := codecCtx.Open(codec, nil); err != nil {
		codecCtx.Free()
		return nil, fmt.Errorf("failed to open codec context: %w", err)
	}

	softwareFrame := astiav.AllocFrame()
	if softwareFrame == nil {
		codecCtx.Free()
		return nil, fmt.Errorf("failed to allocate frame")
	}

	softwareFrame.SetWidth(p.Width)
	softwareFrame.SetHeight(p.Height)
	softwareFrame.SetPixelFormat(astiav.PixelFormat(astiav.PixelFormatYuv420P))

	err = softwareFrame.AllocBuffer(0)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate sorfware buffer: %w", err)
	}

	hardwareFrame := astiav.AllocFrame()

	err = hardwareFrame.AllocHardwareBuffer(hwFramesCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate hardware buffer: %w", err)
	}

	packet := astiav.AllocPacket()
	if packet == nil {
		softwareFrame.Free()
		codecCtx.Free()
		return nil, fmt.Errorf("failed to allocate packet")
	}

	return &encoder{
		codec:       codec,
		codecCtx:    codecCtx,
		hwFramesCtx: hwFramesCtx,
		frame:       softwareFrame,
		hwFrame:     hardwareFrame,
		packet:      packet,
		width:       p.Width,
		height:      p.Height,
		r:           video.ToI420(r),
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

	img, release, err := e.r.Read()
	if err != nil {
		return nil, func() {}, err
	}
	defer release()

	err = e.frame.Data().FromImage(img)
	if err != nil {
		return nil, func() {}, fmt.Errorf("failed to copy image data: %w", err)
	}

	err = e.frame.TransferHardwareData(e.hwFrame)
	if err != nil {
		return nil, func() {}, err
	}

	// Send frame to encoder
	if err := e.codecCtx.SendFrame(e.hwFrame); err != nil {
		return nil, func() {}, fmt.Errorf("failed to send frame: %w", err)
	}

	for {
		if err = e.codecCtx.ReceivePacket(e.packet); err != nil {
			if errors.Is(err, astiav.ErrEof) || errors.Is(err, astiav.ErrEagain) {
				continue
			}
			return nil, func() {}, fmt.Errorf("failed to receive packet: %w", err)
		}
		break
	}

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
