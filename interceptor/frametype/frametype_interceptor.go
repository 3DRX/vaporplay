package frametype

import (
	"log/slog"

	"github.com/pion/interceptor"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

type FrameTypeInterceptorFactory struct{}

func (f *FrameTypeInterceptorFactory) NewInterceptor(id string) (interceptor.Interceptor, error) {
	return &FrameTypeInterceptor{}, nil
}

func NewFrameTypeInterceptor() (*FrameTypeInterceptorFactory, error) {
	return &FrameTypeInterceptorFactory{}, nil
}

type FrameTypeData struct {
	FrameType FrameTypeEnum
	Start     bool
	FrameID   uint64
}

const AttributesKey = "frameTypeData"

type FrameTypeEnum int

const (
	FrameTypeUnknown FrameTypeEnum = iota
	FrameTypeKeyFrame
	FrameTypeDeltaFrame
	FrameTypeSEI
)

type FrameTypeInterceptor struct {
	interceptor.NoOp
}

func (i *FrameTypeInterceptor) BindLocalStream(
	info *interceptor.StreamInfo,
	writer interceptor.RTPWriter,
) interceptor.RTPWriter {
	var nalUnitType, S, h264NalUnitType byte
	var frameID uint64
	switch info.MimeType {
	case webrtc.MimeTypeH264:
		return interceptor.RTPWriterFunc(
			func(header *rtp.Header, payload []byte, attributes interceptor.Attributes) (int, error) {
				frameTypeData := FrameTypeData{
					FrameType: FrameTypeUnknown,
					FrameID:   frameID,
				}
				if header.Marker {
					frameID++
				}
				nalUnitType = payload[0] & 0x1F
				if 1 <= nalUnitType && nalUnitType <= 23 {
					if nalUnitType == 6 {
						frameTypeData.FrameType = FrameTypeSEI
					} else {
						frameTypeData.FrameType = FrameTypeDeltaFrame
						frameTypeData.Start = true
					}
				} else if nalUnitType == 28 {
					S = payload[1] >> 7
					h264NalUnitType = payload[1] & 0x1F
					frameTypeData.Start = S == 1
					switch h264NalUnitType {
					case 5:
						frameTypeData.FrameType = FrameTypeKeyFrame
					default:
						frameTypeData.FrameType = FrameTypeDeltaFrame
					}
				} else if nalUnitType == 24 {
					// For STAP-A, we only read the first NAL unit header,
					// and assume the rest are the same as the first one.
					S = payload[3] >> 7
					h264NalUnitType = payload[3] & 0x1F
					frameTypeData.Start = S == 1
					switch h264NalUnitType {
					case 5:
						frameTypeData.FrameType = FrameTypeKeyFrame
					default:
						frameTypeData.FrameType = FrameTypeDeltaFrame
					}
				} else {
					slog.Warn("FrameTypeInterceptor nalUnitType not supported", "nalUnitType", nalUnitType)
				}

				attributes.Set(AttributesKey, frameTypeData)

				return writer.Write(header, payload, attributes)
			},
		)
	default:
		return interceptor.RTPWriterFunc(
			func(header *rtp.Header, payload []byte, attributes interceptor.Attributes) (int, error) {
				return writer.Write(header, payload, attributes)
			},
		)
	}

}
