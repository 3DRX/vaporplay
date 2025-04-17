package peerconnection

import (
	"errors"
	"image"
	"log/slog"

	"github.com/asticode/go-astiav"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v4/pkg/media/samplebuilder"
)

type VideoDecoder struct {
	sampleBuilder *samplebuilder.SampleBuilder

	codecCreated bool

	pkt         *astiav.Packet
	frame       *astiav.Frame
	decCodec    *astiav.Codec
	decCodecCtx *astiav.CodecContext
}

func newVideoDecoder() *VideoDecoder {
	return &VideoDecoder{
		sampleBuilder: samplebuilder.New(200, &codecs.H264Packet{}, 90000),
		codecCreated:  false,
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
	if s.decCodec = astiav.FindDecoder(astiav.CodecID(astiav.CodecIDH264)); s.decCodec == nil {
		panic("failed to find decoder")
	}
	if s.decCodecCtx = astiav.AllocCodecContext(s.decCodec); s.decCodecCtx == nil {
		panic("failed to allocate codec context")
	}
	if err := s.decCodecCtx.Open(s.decCodec, nil); err != nil {
		panic("failed to open codec context")
	}
	s.codecCreated = true
}
