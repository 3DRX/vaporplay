// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package flexfec

import (
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/pion/interceptor"
	"github.com/pion/rtp"
)

// FecInterceptor implements FlexFec.
type FecInterceptor struct {
	interceptor.NoOp
	flexFecEncoder     FlexEncoder
	packetBuffer       []rtp.Packet
	minNumMediaPackets uint32
	payloadType        uint8
	ssrc               uint32

	fecBytes  uint64
	startTime time.Time
	mu        sync.Mutex

	//新增字段，用来动态调整FEC的参数
	protectionMethod *ProtectionMethod
	params           *ProtectionParams
	packetsPerFrame  *MovingAverage
	lastStatsUpdate  time.Time
	currentLossRate  float32
	currentRTT       int64
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
		minNumMediaPackets: 3,
	}
	// 添加
	interceptor.protectionMethod = ProtectionMethod()
	interceptor.params = &ProtectionParams{
		// 初始值
	}
	interceptor.packetsPerFrame = MovingAverage(0.9999)
	interceptor.lastStatsUpdate = time.Now()

	instance = interceptor

	return interceptor, nil
}

// BindLocalStream lets you modify any outgoing RTP packets. It is called once for per LocalStream. The returned method
// will be called once per rtp packet.
func (r *FecInterceptor) BindLocalStream(
	info *interceptor.StreamInfo, writer interceptor.RTPWriter,
) interceptor.RTPWriter {
	if r.payloadType != 0 {
		info.PayloadTypeForwardErrorCorrection = r.payloadType
	}
	if r.ssrc != 0 {
		info.SSRCForwardErrorCorrection = r.ssrc
	}
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

			// Send the media RTP packet
			result, err := writer.Write(header, payload, attributes)

			// Send the FEC packets
			var fecPackets []rtp.Packet
			// for frame smaller than 5 packets, encode FEC with next frame
			if header.Marker && len(r.packetBuffer) >= int(r.minNumMediaPackets) {
				// 计算应该生成多少个 FEC 包
				numFECPackets := r.calculateNumFECPackets()
				fecPackets = r.flexFecEncoder.EncodeFec(r.packetBuffer, numFECPackets)

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

func (r *FecInterceptor) SetFecPayloadType(payloadType uint8) {
	r.payloadType = payloadType
}

func (r *FecInterceptor) SetFecSSRC(ssrc uint32) {
	r.ssrc = ssrc
}

func SetFecPayloadType(payloadType uint8) {
	if instance != nil {
		instance.SetFecPayloadType(payloadType)
	}
}

func SetFecSSRC(ssrc uint32) {
	if instance != nil {
		instance.SetFecSSRC(ssrc)
	}
}

func GetFECBitrate() float64 {
	if instance == nil {
		return 0
	}
	return instance.GetFECBitRate()
}

// 添加新方法计算需要的 FEC 包数量
func (r *FecInterceptor) calculateNumFECPackets() uint32 {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 1. 检查丢包率是否过低无需保护
	if r.currentLossRate < 0.01 { // 低于1%时不添加保护
		return 0
	}

	// 2. 更新保护参数
	// 假设 r.protectionMethod 是 ProtectionCalculator 的实例
	r.protectionMethod.UpdateRTT(r.currentRTT)
	r.protectionMethod.UpdatePacketLoss(r.currentLossRate)

	// 3. 计算保护因子
	deltaFactor, _ := r.protectionMethod.CalculateProtectionFactors()

	// 4. 基于保护因子计算 FEC 包数量
	mediaPackets := len(r.packetBuffer)
	fecRatio := float64(deltaFactor) / 255.0
	fecCount := fecRatio * float64(mediaPackets) / (1.0 - fecRatio)

	// 至少生成1个 FEC 包
	result := uint32(math.Max(1, math.Round(fecCount)))

	// 限制最大 FEC 包数
	maxFec := uint32(mediaPackets) / 2
	if result > maxFec && maxFec > 0 {
		result = maxFec
	}

	return result
}

// 添加设置网络状态的方法
func (r *FecInterceptor) SetLossRate(lossRate float32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.currentLossRate = lossRate
}

func (r *FecInterceptor) SetRTT(rtt int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.currentRTT = rtt
}
