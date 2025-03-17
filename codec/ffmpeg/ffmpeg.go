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

type hardwareEncoder struct {
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

type VP8Params struct {
	Params
}

func NewVP8VAAPIParams(hardwareDevice string, pixelFormat astiav.PixelFormat) (VP8Params, error) {
	return VP8Params{
		Params: Params{
			codecName:      "vp8_vaapi",
			hardwareDevice: hardwareDevice,
			pixelFormat:    pixelFormat,
		},
	}, nil
}

func (p *VP8Params) RTPCodec() *codec.RTPCodec {
	defaultH264Codec := codec.NewRTPVP8Codec(90000)
	return defaultH264Codec
}

func (p *VP8Params) BuildVideoEncoder(r video.Reader, property prop.Media) (codec.ReadCloser, error) {
	readCloser, err := newHardwareEncoder(r, property, p.Params)
	if err != nil {
		slog.Error("failed to create new encoder", "error", err)
		return nil, err
	}
	slog.Info("sucsessfully created new encoder")
	return readCloser, nil
}

type H264Params struct {
	Params
}

func NewH264NVENCParams(hardwareDevice string, pixelFormat astiav.PixelFormat) (H264Params, error) {
	return H264Params{
		Params: Params{
			codecName:      "h264_nvenc",
			hardwareDevice: hardwareDevice,
			pixelFormat:    pixelFormat,
		},
	}, nil
}

func NewH264VAAPIParams(hardwareDevice string, pixelFormat astiav.PixelFormat) (H264Params, error) {
	return H264Params{
		Params: Params{
			codecName:      "h264_vaapi",
			hardwareDevice: hardwareDevice,
			pixelFormat:    pixelFormat,
		},
	}, nil
}

// RTPCodec represents the codec metadata
func (p *H264Params) RTPCodec() *codec.RTPCodec {
	defaultH264Codec := codec.NewRTPH264Codec(90000)
	return defaultH264Codec
}

func (p *H264Params) BuildVideoEncoder(r video.Reader, property prop.Media) (codec.ReadCloser, error) {
	readCloser, err := newHardwareEncoder(r, property, p.Params)
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

func NewH265NVENCParams(hardwareDevice string, pixelFormat astiav.PixelFormat) (H265Params, error) {
	return H265Params{
		Params: Params{
			codecName:      "hevc_nvenc",
			hardwareDevice: hardwareDevice,
			pixelFormat:    pixelFormat,
		},
	}, nil
}

func NewH265VAAPIParams(hardwareDevice string, pixelFormat astiav.PixelFormat) (H265Params, error) {
	return H265Params{
		Params: Params{
			codecName:      "hevc_vaapi",
			hardwareDevice: hardwareDevice,
			pixelFormat:    pixelFormat,
		},
	}, nil
}

func (p *H265Params) RTPCodec() *codec.RTPCodec {
	defaultH265Codec := codec.NewRTPH265Codec(90000)
	return defaultH265Codec
}

func (p *H265Params) BuildVideoEncoder(r video.Reader, property prop.Media) (codec.ReadCloser, error) {
	readCloser, err := newHardwareEncoder(r, property, p.Params)
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

func NewAV1NVENCParams(hardwareDevice string, pixelFormat astiav.PixelFormat) (AV1Params, error) {
	return AV1Params{
		Params: Params{
			codecName:      "av1_nvenc",
			hardwareDevice: hardwareDevice,
			pixelFormat:    pixelFormat,
		},
	}, nil
}

func (p *AV1Params) RTPCodec() *codec.RTPCodec {
	defaultAV1Codec := codec.NewRTPAV1Codec(90000)
	return defaultAV1Codec
}

func (p *AV1Params) BuildVideoEncoder(r video.Reader, property prop.Media) (codec.ReadCloser, error) {
	readCloser, err := newHardwareEncoder(r, property, p.Params)
	if err != nil {
		slog.Error("failed to create new encoder", "error", err)
		return nil, err
	}
	slog.Info("sucsessfully created new encoder")
	return readCloser, nil
}

func newHardwareEncoder(r video.Reader, p prop.Media, params Params) (*hardwareEncoder, error) {
	if p.FrameRate == 0 {
		slog.Warn(fmt.Sprintf("frame rate is 0, setting to %f", params.FrameRate))
		p.FrameRate = params.FrameRate
	}
	slog.Info("creating new encoder", "params", params, "props", p)
	astiav.SetLogLevel(astiav.LogLevel(astiav.LogLevelWarning))

	var hardwareDeviceType astiav.HardwareDeviceType
	switch params.codecName {
	case "h264_nvenc", "hevc_nvenc", "av1_nvenc":
		hardwareDeviceType = astiav.HardwareDeviceType(astiav.HardwareDeviceTypeCUDA)
	case "vp8_vaapi", "h264_vaapi", "hevc_vaapi":
		hardwareDeviceType = astiav.HardwareDeviceType(astiav.HardwareDeviceTypeVAAPI)
	}

	hwDevice, err := astiav.CreateHardwareDeviceContext(
		hardwareDeviceType,
		params.hardwareDevice,
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
	codecCtx.SetBitRate(int64(params.BitRate))
	codecCtx.SetGopSize(params.KeyFrameInterval)
	switch params.codecName {
	case "h264_nvenc", "hevc_nvenc", "av1_nvenc":
		codecCtx.SetPixelFormat(astiav.PixelFormat(astiav.PixelFormatCuda))
	case "vp8_vaapi", "h264_vaapi", "hevc_vaapi":
		codecCtx.SetPixelFormat(astiav.PixelFormat(astiav.PixelFormatVaapi))
	}
	codecOptions := codecCtx.PrivateData().Options()
	switch params.codecName {
	case "av1_nvenc":
		codecCtx.SetProfile(astiav.Profile(astiav.ProfileAv1Main))
		codecOptions.Set("tier", "0", 0)
	case "h264_vaapi":
		codecCtx.SetProfile(astiav.Profile(astiav.ProfileH264Main))
		codecOptions.Set("profile", "main", 0)
		codecOptions.Set("level", "1", 0)
	case "hevc_vaapi":
		codecCtx.SetProfile(astiav.Profile(astiav.ProfileHevcMain))
		codecOptions.Set("profile", "main", 0)
		codecOptions.Set("tier", "main", 0)
		codecOptions.Set("level", "1", 0)
	}
	switch params.codecName {
	case "h264_nvenc", "hevc_nvenc", "av1_nvenc":
		codecOptions.Set("forced-idr", "1", 0)
		codecOptions.Set("zerolatency", "1", 0)
		codecOptions.Set("delay", "0", 0)
		codecOptions.Set("tune", "ull", 0)
		codecOptions.Set("preset", "p1", 0)
		codecOptions.Set("rc", "cbr", 0)
	case "vp8_vaapi", "h264_vaapi", "hevc_vaapi":
		codecOptions.Set("rc_mode", "CBR", 0)
	}

	// Create hardware frames context
	hwFramesCtx := astiav.AllocHardwareFramesContext(hwDevice)
	if hwFramesCtx == nil {
		hwDevice.Free()
		return nil, fmt.Errorf("failed to allocate hw frames context")
	}

	// Set hardware frames context parameters
	hwFramesCtx.SetWidth(p.Width)
	hwFramesCtx.SetHeight(p.Height)
	switch params.codecName {
	case "h264_nvenc", "hevc_nvenc", "av1_nvenc":
		hwFramesCtx.SetHardwarePixelFormat(astiav.PixelFormat(astiav.PixelFormatCuda))
	case "vp8_vaapi", "h264_vaapi", "hevc_vaapi":
		hwFramesCtx.SetHardwarePixelFormat(astiav.PixelFormat(astiav.PixelFormatVaapi))
	}
	hwFramesCtx.SetSoftwarePixelFormat(params.pixelFormat)

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
	softwareFrame.SetPixelFormat(params.pixelFormat)

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
	// go StatsThread(statsItemChan)

	return &hardwareEncoder{
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

func (e *hardwareEncoder) Controller() codec.EncoderController {
	return e
}

func (e *hardwareEncoder) Read() ([]byte, func(), error) {
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

	// e.statsItemChan <- StatsItem{
	// 	FrameSize: e.packet.Size(),
	// }
	data := make([]byte, e.packet.Size())
	copy(data, e.packet.Data())
	e.packet.Unref()

	return data, func() {}, nil
}

// ForceKeyFrame forces the next frame to be encoded as a keyframe
func (e *hardwareEncoder) ForceKeyFrame() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	slog.Info("forcing key frame")
	e.nextIsKeyFrame = true
	return nil
}

func (e *hardwareEncoder) SetBitRate(bitrate int) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.codecCtx.SetBitRate(int64(bitrate))
	return nil
}

func (e *hardwareEncoder) Close() error {
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
