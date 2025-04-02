// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package flexfec

import (
	"log/slog"
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
	// 动态 FEC 相关字段
	protectionMethod     *VCMProtectionMethod // 保护方法计算器
	packetsPerFrame      *MovingAverage       // 每帧包数的移动平均
	packetsPerFrameKey   *MovingAverage       // 关键帧包数的移动平均
	currentLossRate      float32              // 当前丢包率
	currentRTT           int64                // 当前 RTT (ms)
	currentBitRate       float32              // 当前比特率 (bps)
	frameRate            float32              // 帧率
	codecWidth           int                  // 编码宽度
	codecHeight          int                  // 编码高度
	numLayers            int                  // 编码层数
	framePacketCount     int                  // 当前帧的包计数
	isFirstPacketInFrame bool
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

		// 初始化动态 FEC 相关字段
		protectionMethod:     NewVCMProtectionMethod(),
		packetsPerFrame:      NewMovingAverage(0.9999),
		packetsPerFrameKey:   NewMovingAverage(0.9999),
		currentLossRate:      0.0,
		currentRTT:           0, // 默认
		currentBitRate:       0,
		frameRate:            0,
		codecWidth:           0,
		codecHeight:          0,
		numLayers:            1,
		isFirstPacketInFrame: true,
	}

	instance = interceptor

	return interceptor, nil
}

// IsKeyFrame 检测 RTP 包是否为关键帧
func IsKeyFrame(packet rtp.Packet) bool {
	if len(packet.Payload) < 2 {
		return false
	}
	// H.264的NAL头的前字节格式: |F(1)|NRI(2)|Type(5)|
	h264NalType := packet.Payload[0] & 0x1F
	if h264NalType >= 1 && h264NalType <= 23 {
		// 合法的H.264 NAL类型
		// Type=5表示IDR帧(关键帧)
		return h264NalType == 5
	}

	hasExtendedControlBits := (packet.Payload[0] & 0x10) != 0
	// 计算VP8 Payload Descriptor的长度
	vpxHeaderSize := 1
	if hasExtendedControlBits && len(packet.Payload) > 1 {
		vpxHeaderSize++
		if (packet.Payload[1] & 0x80) != 0 { // I位 - PictureID present
			if len(packet.Payload) > vpxHeaderSize && (packet.Payload[vpxHeaderSize]&0x80) != 0 {
				// M位设置，PictureID为2字节
				vpxHeaderSize += 2
			} else {
				vpxHeaderSize++
			}
		}
		if (packet.Payload[1] & 0x20) != 0 { // L位 - TL0PICIDX present
			vpxHeaderSize++
		}
		if (packet.Payload[1]&0x40) != 0 || (packet.Payload[1]&0x10) != 0 {
			// T位或K位设置 - TID/KID present
			vpxHeaderSize++
		}
	}
	// 确保有足够数据读取VP8帧头
	if len(packet.Payload) > vpxHeaderSize {
		// 检查VP8的帧类型位
		// 在VP8数据的第一个字节，最低位为0表示关键帧
		return (packet.Payload[vpxHeaderSize] & 0x01) == 0
	}
	return false
}

// isKeyFrame 检测是否为关键帧
func isKeyFrame(packet rtp.Packet) bool {
	if len(packet.Payload) == 0 {
		return false
	}
	// 直接调用统一的关键帧检测函数
	return IsKeyFrame(packet)
}

// isCurrentFrameKeyFrame 检测当前缓存帧是否为关键帧
func (r *FecInterceptor) isCurrentFrameKeyFrame() bool {
	if len(r.packetBuffer) == 0 {
		return false
	}
	return IsKeyFrame(r.packetBuffer[0])
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
			r.packetBuffer = append(r.packetBuffer, rtp.Packet{ //添加到缓冲区
				Header:  *header,
				Payload: payload,
			})

			// 跟踪帧包计数
			r.mu.Lock()
			isFirstPacket := r.isFirstPacketInFrame
			r.framePacketCount++
			r.isFirstPacketInFrame = false
			r.mu.Unlock()

			// // An example of how to get the frame type data from the attributes
			// frameTypeData, ok := attributes.Get(frametype.AttributesKey).(frametype.FrameTypeData)
			// if ok {
			// 	slog.Info("FecInterceptor", "frameTypeData", frameTypeData)
			// }

			// 检测关键帧并更新统计
			if isFirstPacket {
				isKey := isKeyFrame(r.packetBuffer[len(r.packetBuffer)-1])
				if isKey {
					// 记录上一帧是关键帧，为后续计算做准备
					slog.Debug("it's Key Frame")
				}
			}

			// Send the media RTP packet
			result, err := writer.Write(header, payload, attributes)

			// Send the FEC packets
			var fecPackets []rtp.Packet
			// for frame smaller than 5 packets, encode FEC with next frame
			if len(r.packetBuffer) >= int(r.minNumMediaPackets) {
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

// NumFecPackets 根据媒体包数量和保护因子计算需要的FEC包数量
func NumFecPackets(numMediaPackets int, protectionFactor uint8) uint32 {
	// 定点数计算，保持与C++版本一致
	numFecPackets := (numMediaPackets*int(protectionFactor) + (1 << 7)) >> 8

	// 如果需要保护但计算结果为0，至少生成1个FEC包
	if protectionFactor > 0 && numFecPackets == 0 {
		numFecPackets = 1
	}

	// 确保FEC包数不超过媒体包数
	if numFecPackets > numMediaPackets {
		numFecPackets = numMediaPackets
	}

	return uint32(numFecPackets)
}

// calculateNumFECPackets 根据当前网络状态计算所需FEC包数
func (r *FecInterceptor) calculateNumFECPackets() uint32 {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 1. 媒体包数量过少时不生成FEC
	mediaPackets := len(r.packetBuffer)
	if mediaPackets < int(r.minNumMediaPackets) {
		return 0
	}

	// 2. 丢包率很低时不需要保护
	if r.currentLossRate < 0.005 { // 0.5%
		return 0
	}

	// 3. 构建保护参数
	params := &VCMProtectionParameters{
		rtt:                r.currentRTT,
		lossPr:             r.currentLossRate,
		bitRate:            r.currentBitRate,
		frameRate:          r.frameRate,
		packetsPerFrame:    r.packetsPerFrame.Filtered(),
		packetsPerFrameKey: r.packetsPerFrameKey.Filtered(),
		codecWidth:         r.codecWidth,
		codecHeight:        r.codecHeight,
		numLayers:          r.numLayers,
	}

	// 4. 检查比特率是否太低不适合FEC
	if r.protectionMethod.BitRateTooLowForFec(params) {
		return 0
	}
	// 5. 更新保护参数
	r.protectionMethod.UpdateParameters(params)
	// 6. 获取保护因子
	deltaFactor, keyFactor := r.protectionMethod.CalculateProtectionFactors()

	// 7. 判断是否为关键帧
	isKeyFrame := r.isCurrentFrameKeyFrame()

	// 8. 选择合适的保护因子
	factor := deltaFactor
	if isKeyFrame {
		factor = keyFactor
	}

	// 9. 简单地用 NumFecPackets 计算所需的 FEC 包数量
	return NumFecPackets(mediaPackets, factor)
}

// 一些设置当前属性的函数
func (r *FecInterceptor) SetLossRate(lossRate float32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.currentLossRate = lossRate
}

// 设置当前RTT
func (r *FecInterceptor) SetRTT(rtt int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.currentRTT = rtt
}

// 设置比特率
func (r *FecInterceptor) SetBitRate(bitRate float32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.currentBitRate = bitRate
}

// 设置视频参数
func (r *FecInterceptor) SetVideoParams(width, height int, frameRate float32, numLayers int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.codecWidth = width
	r.codecHeight = height
	r.frameRate = frameRate
	r.numLayers = numLayers
}

// 全局访问方法
func SetLossRate(lossRate float32) {
	if instance != nil {
		instance.SetLossRate(lossRate)
	}
}

func SetRTT(rtt int64) {
	if instance != nil {
		instance.SetRTT(rtt)
	}
}

func SetBitRate(bitRate float32) {
	if instance != nil {
		instance.SetBitRate(bitRate)
	}
}

func SetVideoParams(width, height int, frameRate float32, numLayers int) {
	if instance != nil {
		instance.SetVideoParams(width, height, frameRate, numLayers)
	}
}
