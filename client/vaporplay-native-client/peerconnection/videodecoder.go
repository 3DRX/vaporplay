package peerconnection

import (
	"time"

	"github.com/asticode/go-astiav"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v4/pkg/media/samplebuilder"
)

type VideoDecoder struct {
	sampleBuilder  *samplebuilder.SampleBuilder
	videoTimestamp time.Duration

	lastVideoTimestamp uint32
	codecCreated       bool
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

		// Decode VP8 frame
		// codecError := C.decode_frame(&s.codecCtx, (*C.uint8_t)(&sample.Data[0]), C.size_t(len(sample.Data)))
		// if codecError != 0 {
		// 	slog.Error("Decode error", "errorCode", codecError)
		// 	continue
		// }
		// // Get decoded frames
		// var iter C.vpx_codec_iter_t
		// img := C.vpx_codec_get_frame(&s.codecCtx, &iter)
		// if img == nil {
		// 	slog.Error("Failed to get decoded frame")
		// 	continue
		// }
		// var ros_img sensor_msgs_msg.Image
		// var ros_img_c C.sensor_msgs__msg__Image
		// C.vpx_to_ros_image(img, &ros_img_c)
		// sensor_msgs_msg.ImageTypeSupport.AsGoStruct(&ros_img, unsafe.Pointer(&ros_img_c))
		// C.cleanup_ros_image(&ros_img_c)
		// s.imgChan <- &ros_img
	}
}

func (s *VideoDecoder) Init(width, height int) {
	// if errCode := C.init_decoder(&s.codecCtx, C.uint(width), C.uint(height)); errCode != 0 {
	// 	slog.Error("failed to initialize decoder", "error", errCode)
	// }
	astiav.SetLogLevel(astiav.LogLevel(astiav.LogLevelDebug))
	s.codecCreated = true
}
