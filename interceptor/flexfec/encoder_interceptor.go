// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package flexfec

import (
	"log/slog"
	"sync"
	"time"

	"github.com/3DRX/vaporplay/interceptor/frametype"
	"github.com/3DRX/vaporplay/utils"
	"github.com/pion/interceptor"
	"github.com/pion/rtp"
)

// FecInterceptor implements FlexFec.
type FecInterceptor struct {
	interceptor.NoOp
	flexFecEncoder     FlexEncoder
	packetBuffer       []rtp.Packet
	minNumMediaPackets uint32

	fecBytes  uint64
	startTime time.Time
	mu        sync.Mutex
}

var instance *FecInterceptor

// FecOption can be used to set initial options on Fec encoder interceptors.
type FecOption func(d *FecInterceptor) error

// FecInterceptorFactory creates new FecInterceptors.
type FecInterceptorFactory struct {
	opts []FecOption
}

// NewFecInterceptor returns a new Fec interceptor factory.
func NewFecInterceptor(opts ...FecOption) (*FecInterceptorFactory, error) {
	return &FecInterceptorFactory{opts: opts}, nil
}

// NewInterceptor constructs a new FecInterceptor.
func (r *FecInterceptorFactory) NewInterceptor(_ string) (interceptor.Interceptor, error) {
	interceptor := &FecInterceptor{
		packetBuffer:       make([]rtp.Packet, 0),
		minNumMediaPackets: 5,
	}

	instance = interceptor

	return interceptor, nil
}

// BindLocalStream lets you modify any outgoing RTP packets. It is called once for per LocalStream. The returned method
// will be called once per rtp packet.
func (r *FecInterceptor) BindLocalStream(
	info *interceptor.StreamInfo, writer interceptor.RTPWriter,
) interceptor.RTPWriter {
	slog.Info(
		"FecInterceptor BindLocalStream",
		"ssrc",
		info.SSRC,
		"fec",
		info.SSRCForwardErrorCorrection,
		"mimeType",
		info.MimeType,
		"payloadType",
		info.PayloadType,
		"payloadTypeFec",
		info.PayloadTypeForwardErrorCorrection,
	)
	// Chromium supports version flexfec-03 of existing draft, this is the one we will configure by default
	// although we should support configuring the latest (flexfec-20) as well.
	r.flexFecEncoder = NewFlexEncoder03(info.PayloadTypeForwardErrorCorrection, info.SSRCForwardErrorCorrection)

	return interceptor.RTPWriterFunc(
		func(header *rtp.Header, payload []byte, attributes interceptor.Attributes) (int, error) {
			r.packetBuffer = append(r.packetBuffer, rtp.Packet{
				Header:  *header,
				Payload: payload,
			})

			// An example of how to get the frame type data from the attributes
			frameTypeData, ok := attributes.Get(frametype.AttributesKey).(utils.FrameTypeData)
			if ok {
				if frameTypeData.Start {
					if frameTypeData.FrameType == utils.FrameTypeKeyFrame {
						slog.Info("FecInterceptor key frame!!!!!!!!!!!")
					}
				}
			}

			// Send the media RTP packet
			result, err := writer.Write(header, payload, attributes)

			// Send the FEC packets
			var fecPackets []rtp.Packet
			if len(r.packetBuffer) >= int(r.minNumMediaPackets) {
				fecPackets = r.flexFecEncoder.EncodeFec(r.packetBuffer, 1)

				for i := range fecPackets {
					size := uint64(fecPackets[i].Header.MarshalSize() + len(fecPackets[i].Payload))
					fecResult, fecErr := writer.Write(&(fecPackets[i].Header), fecPackets[i].Payload, attributes)

					if fecErr != nil && fecResult == 0 {
						break
					} else {
						r.mu.Lock()
						r.fecBytes += size
						if r.startTime.IsZero() {
							r.startTime = time.Now()
						}
						r.mu.Unlock()
					}
				}
				// Reset the packet buffer now that we've sent the corresponding FEC packets.
				r.packetBuffer = nil
			}

			return result, err
		},
	)
}

func (r *FecInterceptor) GetFECBitRate() float64 {
	r.mu.Lock()
	startTime := r.startTime
	fecBytes := r.fecBytes
	r.mu.Unlock()

	if startTime.IsZero() {
		return 0
	}
	now := time.Now()
	duration := now.Sub(startTime).Seconds()
	if duration == 0 {
		return 0
	}
	bitrate := (float64(fecBytes) * 8) / duration
	r.mu.Lock()
	r.fecBytes -= fecBytes
	r.startTime = now
	r.mu.Unlock()
	return bitrate
}

func GetFECBitrate() float64 {
	if instance == nil {
		return 0
	}
	return instance.GetFECBitRate()
}
