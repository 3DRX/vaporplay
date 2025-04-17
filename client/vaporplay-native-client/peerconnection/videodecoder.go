package peerconnection

import (
	"errors"
	"image"
	"log/slog"

	"github.com/3DRX/vaporplay/config"
	"github.com/asticode/go-astiav"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v4/pkg/media/samplebuilder"
)

type VideoDecoder struct {
	sampleBuilder *samplebuilder.SampleBuilder

	codec        string
	codecCreated bool

	pkt         *astiav.Packet
	frame       *astiav.Frame
	decCodec    *astiav.Codec
	decCodecCtx *astiav.CodecContext
}

func newVideoDecoder(codecConfig config.CodecConfig) *VideoDecoder {
	maxLate := uint16(200)
	sampleRate := uint32(90000)
	switch codecConfig.Codec {
	case "av1_nvenc":
		return &VideoDecoder{
			sampleBuilder: samplebuilder.New(maxLate, &codecs.AV1Depacketizer{}, sampleRate),
			codecCreated:  false,
			codec:         codecConfig.Codec,
		}
	case "hevc_nvenc":
		return &VideoDecoder{
			sampleBuilder: samplebuilder.New(maxLate, &codecs.H265Packet{}, sampleRate),
			codecCreated:  false,
			codec:         codecConfig.Codec,
		}
	case "h264_nvenc":
		return &VideoDecoder{
			sampleBuilder: samplebuilder.New(maxLate, &codecs.H264Packet{}, sampleRate),
			codecCreated:  false,
			codec:         codecConfig.Codec,
		}
	default:
		panic("unsupported codec")
	}
}

func (s *VideoDecoder) Close() {
	if s.codecCreated {
		// TODO close codec
	}
}

func (s *VideoDecoder) PushPacket(rtpPacket *rtp.Packet) {
	s.sampleBuilder.Push(rtpPacket)

	for {
		sample := s.sampleBuilder.Pop()
		if sample == nil {
			return
		}

		s.pkt.FromData(sample.Data)
		if err := s.decCodecCtx.SendPacket(s.pkt); err != nil {
			slog.Error("sending packet failed", "error", err)
			return
		}

		for {
			if err := s.decCodecCtx.ReceiveFrame(s.frame); err != nil {
				if errors.Is(err, astiav.ErrEof) || errors.Is(err, astiav.ErrEagain) {
					continue
				}
				slog.Error("receiving frame failed", "error", err)
				return
			}
			break
		}

		dst := &image.RGBA{}
		s.frame.Data().ToImage(dst)
		slog.Info("decoded frame", "width", s.frame.Width(), "height", s.frame.Height())
	}
}

func (s *VideoDecoder) Init() {
	astiav.SetLogLevel(astiav.LogLevel(astiav.LogLevelWarning))

	s.pkt = astiav.AllocPacket()
	s.frame = astiav.AllocFrame()
	switch s.codec {
	case "av1_nvenc":
		// FIXME: wait til https://github.com/asticode/go-astiav/pull/146 is merged
		panic("av1 decoder not implemented yet")
		// if s.decCodec = astiav.FindDecoder(astiav.CodecID(astiav.CodecIDH264)); s.decCodec == nil {
		// 	panic("failed to find decoder")
		// }
	case "hevc_nvenc":
		if s.decCodec = astiav.FindDecoder(astiav.CodecID(astiav.CodecIDHevc)); s.decCodec == nil {
			panic("failed to find decoder")
		}
	case "h264_nvenc":
		if s.decCodec = astiav.FindDecoder(astiav.CodecID(astiav.CodecIDH264)); s.decCodec == nil {
			panic("failed to find decoder")
		}
	default:
		panic("unsupported codec")
	}
	if s.decCodecCtx = astiav.AllocCodecContext(s.decCodec); s.decCodecCtx == nil {
		panic("failed to allocate codec context")
	}
	if err := s.decCodecCtx.Open(s.decCodec, nil); err != nil {
		panic("failed to open codec context")
	}
	s.codecCreated = true
}
