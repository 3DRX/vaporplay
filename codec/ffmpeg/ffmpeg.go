package ffmpeg

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"

	"github.com/asticode/go-astiav"
	"github.com/pion/mediadevices/pkg/codec"
	"github.com/pion/mediadevices/pkg/io/video"
	"github.com/pion/mediadevices/pkg/prop"
)

type encoder struct {
	codec          *astiav.Codec
	codecCtx       *astiav.CodecContext
	hwFramesCtx    *astiav.HardwareFramesContext
	frame          *astiav.Frame
	hwFrame        *astiav.Frame
	packet         *astiav.Packet
	width          int
	height         int
	r              video.Reader
	nextIsKeyFrame bool

	// for stats
	statsItemChan chan StatsItem

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

func (p *H264Params) BuildVideoEncoder(r video.Reader, property prop.Media) (codec.ReadCloser, error) {
	readCloser, err := newEncoder(r, property, p.Params)
	if err != nil {
		slog.Error("failed to create new encoder", "error", err)
		return nil, err
	}
	slog.Info("sucsessfully created new encoder")
	return readCloser, nil
}

type H265Params struct {
	Params
}

func NewH265Params() (H265Params, error) {
	return H265Params{
		Params: Params{
			codecName: "hevc_nvenc",
		},
	}, nil
}

func (p *H265Params) RTPCodec() *codec.RTPCodec {
	return codec.NewRTPH265Codec(90000)
}

func (p *H265Params) BuildVideoEncoder(r video.Reader, property prop.Media) (codec.ReadCloser, error) {
	readCloser, err := newEncoder(r, property, p.Params)
	if err != nil {
		slog.Error("failed to create new encoder", "error", err)
		return nil, err
	}
	slog.Info("sucsessfully created new encoder")
	return readCloser, nil
}

type AV1Params struct {
	Params
}

func NewAV1Params() (AV1Params, error) {
	return AV1Params{
		Params: Params{
			codecName: "av1_nvenc",
		},
	}, nil
}

func (p *AV1Params) RTPCodec() *codec.RTPCodec {
	return codec.NewRTPAV1Codec(90000)
}

func (p *AV1Params) BuildVideoEncoder(r video.Reader, property prop.Media) (codec.ReadCloser, error) {
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
		slog.Warn("frame rate is 0, setting to 90")
		p.FrameRate = 90
	}
	slog.Info("creating new encoder", "params", params, "props", p)
	astiav.SetLogLevel(astiav.LogLevel(astiav.LogLevelWarning))

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
	codecCtx.SetBitRate(int64(params.BitRate))
	codecCtx.SetGopSize(params.KeyFrameInterval)
	codecOptions := codecCtx.PrivateData().Options()
	switch params.codecName {
	case "av1_nvenc":
		codecCtx.SetProfile(astiav.Profile(astiav.ProfileAv1Main))
		codecOptions.Set("tier", "0", 0)
	}
	codecOptions.Set("zerolatency", "1", 0)
	codecOptions.Set("delay", "0", 0)
	codecOptions.Set("tune", "ull", 0)
	codecOptions.Set("preset", "p1", 0)
	codecOptions.Set("rc", "cbr", 0)

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
	hwFramesCtx.SetSoftwarePixelFormat(astiav.PixelFormat(astiav.PixelFormatBgra))
	// hwFramesCtx.SetInitialPoolSize(20)

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
	softwareFrame.SetPixelFormat(astiav.PixelFormat(astiav.PixelFormatBgra))

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

	statsItemChan := make(chan StatsItem, 100)
	go StatsThread(statsItemChan)

	return &encoder{
		codec:          codec,
		codecCtx:       codecCtx,
		hwFramesCtx:    hwFramesCtx,
		frame:          softwareFrame,
		hwFrame:        hardwareFrame,
		packet:         packet,
		width:          p.Width,
		height:         p.Height,
		r:              r,
		nextIsKeyFrame: false,
		statsItemChan:  statsItemChan,
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

	if e.nextIsKeyFrame {
		e.frame.SetPictureType(astiav.PictureType(astiav.PictureTypeI))
		e.hwFrame.SetPictureType(astiav.PictureType(astiav.PictureTypeI))
		e.nextIsKeyFrame = false
	} else {
		e.frame.SetPictureType(astiav.PictureType(astiav.PictureTypeNone))
		e.hwFrame.SetPictureType(astiav.PictureType(astiav.PictureTypeNone))
	}

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

	e.statsItemChan <- StatsItem{
		FrameSize: e.packet.Size(),
	}
	data := make([]byte, e.packet.Size())
	copy(data, e.packet.Data())
	e.packet.Unref()

	return data, func() {}, nil
}

// ForceKeyFrame forces the next frame to be encoded as a keyframe
func (e *encoder) ForceKeyFrame() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	slog.Info("forcing key frame")
	e.nextIsKeyFrame = true
	return nil
}

func (e *encoder) SetBitRate(bitrate int) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.codecCtx.SetBitRate(int64(bitrate))
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
	if e.statsItemChan != nil {
		close(e.statsItemChan)
	}
	return nil
}

type StatsItem struct {
	FrameSize int
}

func StatsThread(frameSizeChan chan StatsItem) {
	// open file for writing
	f, err := os.Create("frame_size.csv")
	if err != nil {
		panic(err)
	}
	w := bufio.NewWriter(f)
	w.WriteString("frame_size\n")
	defer f.Close()
	index := 0
	var statsItem StatsItem
	for {
		select {
		case statsItem = <-frameSizeChan:
			if statsItem.FrameSize == 0 {
				// there will be mutiple 0s when the stream is stopped,
				// it's safe to just ignore them
				continue
			}
			// slog.Info("frame size", "size", frameSize)
			_, err := w.WriteString(fmt.Sprintf("%d\n", statsItem.FrameSize))
			if err != nil {
				slog.Error("failed to write frame size to file", "error", err)
			}
			if index%270 == 0 {
				w.Flush()
			}
			index++
		}
	}
}
