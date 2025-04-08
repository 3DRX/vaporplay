package peerconnection

import (
	"github.com/3DRX/vaporplay/interceptor/rfc8888"
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v4"
)

// ConfigureCongestionControlFeedback registers congestion control feedback as
// defined in RFC 8888 (https://datatracker.ietf.org/doc/rfc8888/)
func ConfigureCongestionControlFeedback(mediaEngine *webrtc.MediaEngine, interceptorRegistry *interceptor.Registry) error {
	mediaEngine.RegisterFeedback(webrtc.RTCPFeedback{Type: webrtc.TypeRTCPFBACK, Parameter: "ccfb"}, webrtc.RTPCodecTypeVideo)
	mediaEngine.RegisterFeedback(webrtc.RTCPFeedback{Type: webrtc.TypeRTCPFBACK, Parameter: "ccfb"}, webrtc.RTPCodecTypeAudio)
	generator, err := rfc8888.NewSenderInterceptor()
	if err != nil {
		return err
	}
	interceptorRegistry.Add(generator)

	return nil
}
