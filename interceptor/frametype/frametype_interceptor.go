package frametype

import (
	"github.com/3DRX/vaporplay/utils"
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

const AttributesKey = "frameTypeData"

type FrameTypeInterceptor struct {
	interceptor.NoOp
}

func (i *FrameTypeInterceptor) BindLocalStream(
	info *interceptor.StreamInfo,
	writer interceptor.RTPWriter,
) interceptor.RTPWriter {
	var frameID uint64
	switch info.MimeType {
	case webrtc.MimeTypeH264:
		return interceptor.RTPWriterFunc(
			func(header *rtp.Header, payload []byte, attributes interceptor.Attributes) (int, error) {
				frameTypeData := utils.GetFrameTypeDataFromH264Packet(&rtp.Packet{
					Header:  *header,
					Payload: payload,
				})
				frameTypeData.FrameID = frameID
				if header.Marker {
					frameID++
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
