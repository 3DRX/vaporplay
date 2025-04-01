package flexfec

import (
	"math"
	"sync"
)

const (
	// 尝试和C++版本变量名一致
	kPacketLossMax  = 129
	minProtLevelFec = 85
	kFec            = 2 // 保护方法类型常量
	//比特率/帧 和rrt,低于以下值需要关掉fec depending on `_numLayers` and `_maxFramesFec`.
	// Max bytes/frame for VGA, corresponds to ~140k at 25fps.
	maxBytesPerFrameForFec = 700 //要不要设高一点？
	// Max bytes/frame for CIF and lower: corresponds to ~80k at 25fps.
	maxBytesPerFrameForFecLow = 400
	// Max bytes/frame for frame size larger than VGA, ~200k at 25fps.
	maxBytesPerFrameForFecHigh = 1000
)

// VCMProtectionParameters 存储 FEC 保护参数
type VCMProtectionParameters struct {
	rtt                int64   // 往返时间(ms)
	lossPr             float32 // 丢包率 (0-1)
	bitRate            float32 // 比特率 (bps)
	frameRate          float32 // 帧率 (fps)
	packetsPerFrame    float32 // 每帧的平均包数
	packetsPerFrameKey float32 // 每个关键帧的平均包数
	keyFrameSize       float32 // 关键帧大小
	fecRateDelta       uint8   // Delta帧FEC保护率
	fecRateKey         uint8   // 关键帧FEC保护率
	codecWidth         int     // 编码宽度
	codecHeight        int     // 编码高度
	numLayers          int     // SVC 层数
}

// VCMProtectionMethod 实现 WebRTC FEC 保护因子计算
type VCMProtectionMethod struct {
	mu                  sync.Mutex
	effectivePacketLoss uint8   // 有效丢包率
	protectionFactorK   uint8   // 关键帧保护因子
	protectionFactorD   uint8   // Delta帧保护因子
	scaleProtKey        float32 // 关键帧保护因子缩放系数
	maxPayloadSize      int     // 最大包有效载荷大小
	corrFecCost         float32 // FEC成本校正系数
	pType               int     // 保护方法类型
}

// NewVCMProtectionMethod 创建保护方法计算器
func NewVCMProtectionMethod() *VCMProtectionMethod {
	return &VCMProtectionMethod{
		scaleProtKey:   2.0,
		maxPayloadSize: 1460,
		corrFecCost:    1.0,
		pType:          kFec,
	}
}

// // ProtectionCalculator 计算 FEC 保护参数
// type ProtectionCalculator struct {
// 	mu            sync.Mutex
// 	params        VCMProtectionParameters
// 	statsProvider *StatsProvider

// 	// 重置标志
// 	paramsDirty bool

/*

//VCMProtectionMethod 要写的函数有：
NewVCMProtectionMethod() - 创建保护方法计算器
CalculateProtectionFactors() - 返回计算的保护因子
ProtectionFactor() - 核心算法，计算保护因子
BoostCodeRateKey() - 计算关键帧保护增强系数
ConvertFECRate() - 转换FEC率格式
BitsPerFrame() - 计算每帧比特数
UpdateParameters() - 更新参数并计算保护因子
EffectivePacketLoss() - 计算有效丢包率
BitRateTooLowForFec() - 检查比特率是否太低
MovingAverage 相关函数：

NewMovingAverage() - 创建新的移动平均
Apply() - 应用移动平均
Reset() - 重置移动平均器
Filtered() - 获取过滤后的值
*/

// }

// 上面废弃

// CalculateProtectionFactors 返回计算的Delta帧和关键帧保护因子
func (p *VCMProtectionMethod) CalculateProtectionFactors() (uint8, uint8) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.protectionFactorD, p.protectionFactorK
}

// ProtectionFactor 计算保护因子
func (p *VCMProtectionMethod) ProtectionFactor(params *VCMProtectionParameters) bool {
	// 丢包为0时不需要保护
	packetLoss := uint8(255 * params.lossPr)
	if packetLoss == 0 {
		p.protectionFactorK = 0
		p.protectionFactorD = 0
		return true
	}

	// FEC设置参数:
	// 第一分区保护: ~20%
	firstPartitionProt := uint8(255 * 0.20)

	// 最小保护级别，需要为一个源包/帧生成一个FEC包(在RTP发送器中)
	ratePar1 := uint8(5)
	ratePar2 := uint8(49)

	// 计算空间分辨率相对于参考分辨率(704x576)的比例
	spatialSizeToRef := float32(params.codecWidth*params.codecHeight) / float32(704*576)

	// resolnFac: 这个参数会根据系统大小增加/减少FEC率
	// 使用较小的指数(<1)来控制/软化系统大小效应
	resolnFac := float32(1.0 / math.Pow(float64(spatialSizeToRef), 0.3))

	// 计算每帧比特率
	bitRatePerFrame := p.BitsPerFrame(params)

	// 每帧的平均数据包数(源和fec):
	avgTotPackets := uint8(1.5 + float32(bitRatePerFrame)*1000.0/float32(8.0*p.maxPayloadSize))

	// FEC率参数: 用于P帧和I帧
	var codeRateDelta, codeRateKey uint8 = 0, 0

	// 获取表格索引: FEC保护取决于有效率
	// 速率索引的范围对应于~200k到~8000k的速率(bps)(30fps)
	effRateFecTable := uint16(resolnFac * float32(bitRatePerFrame))

	// 安全计算表格索引，避免可能的数值问题
	var rateIndex int
	if effRateFecTable > uint16(ratePar1) {
		rateIndex = int((effRateFecTable - uint16(ratePar1)) / uint16(ratePar1))
	} else {
		rateIndex = 0
	}

	rateIndexTable := uint8(minInt(maxInt(rateIndex, 0), int(ratePar2)))

	// 将丢包率限制在范围内
	if packetLoss >= kPacketLossMax {
		packetLoss = kPacketLossMax - 1
	}

	// 计算表格索引并安全检查边界
	indexTable := int(rateIndexTable)*kPacketLossMax + int(packetLoss)
	if indexTable >= len(kFecRateTable) {
		indexTable = len(kFecRateTable) - 1
	}
	if indexTable < 0 {
		indexTable = 0
	}

	// P帧的保护因子
	codeRateDelta = kFecRateTable[indexTable]

	// 确保最低保护，如果需要
	if packetLoss > 0 && avgTotPackets > 1 {
		if codeRateDelta < firstPartitionProt {
			codeRateDelta = firstPartitionProt
		}
	}

	// 检查P帧保护限制
	if codeRateDelta >= kPacketLossMax {
		codeRateDelta = kPacketLossMax - 1
	}

	// 对于关键帧:
	// 计算I帧与P帧的包数比例来调整保护因子
	packetFrameDelta := uint8(0.5 + params.packetsPerFrame)
	packetFrameKey := uint8(0.5 + params.packetsPerFrameKey)
	boostKey := p.BoostCodeRateKey(packetFrameDelta, packetFrameKey)

	// 计算关键帧的表格索引
	rateIndexForKey := 0
	boostKeyValue := int(boostKey) * int(effRateFecTable)
	if boostKeyValue > int(ratePar1) {
		rateIndexForKey = 1 + (boostKeyValue-int(ratePar1))/int(ratePar1)
	}

	rateIndexTableKey := uint8(minInt(maxInt(rateIndexForKey, 0), int(ratePar2)))

	// 计算I帧的表格索引
	indexTableKey := int(rateIndexTableKey)*kPacketLossMax + int(packetLoss)
	if indexTableKey >= len(kFecRateTable) {
		indexTableKey = len(kFecRateTable) - 1
	}
	if indexTableKey < 0 {
		indexTableKey = 0
	}

	// I帧保护因子
	codeRateKey = kFecRateTable[indexTableKey]

	// 对关键帧的额外保护
	boostKeyProt := int(p.scaleProtKey * float32(codeRateDelta))
	if boostKeyProt >= kPacketLossMax {
		boostKeyProt = kPacketLossMax - 1
	}

	// 确保I帧保护至少大于P帧保护，且至少与过滤后的丢包率一样高
	// 使用类型安全的max函数
	codeRateKey = uint8(maxInt(int(packetLoss), maxInt(boostKeyProt, int(codeRateKey))))

	// 检查I帧保护量限制：最大127 (kPacketLossMax-1)
	if codeRateKey >= kPacketLossMax {
		codeRateKey = kPacketLossMax - 1
	}

	p.protectionFactorK = codeRateKey
	p.protectionFactorD = codeRateDelta

	// FEC成本校正
	// 校正因子(_corrFecCost)尝试校正这一点，至少对于低速率(小包数)和低保护级别的情况
	numPacketsFl := 1.0 + (float32(bitRatePerFrame)*1000.0/float32(8.0*p.maxPayloadSize) + 0.5)
	estNumFecGen := 0.5 + float32(p.protectionFactorD)*numPacketsFl/255.0

	// 我们减少成本因子(这将降低FEC和混合方法的开销)，而不是保护因子
	p.corrFecCost = 1.0
	if estNumFecGen < 1.1 && p.protectionFactorD < minProtLevelFec {
		p.corrFecCost = 0.5
	}
	if estNumFecGen < 0.9 && p.protectionFactorD < minProtLevelFec {
		p.corrFecCost = 0.0
	}

	return true
}

// BoostCodeRateKey 根据包数比例计算关键帧保护增强系数
func (p *VCMProtectionMethod) BoostCodeRateKey(packetFrameDelta, packetFrameKey uint8) uint8 {
	boostRateKey := uint8(2)
	// 默认：比例按比例放大I帧的FEC保护
	ratio := uint8(1)

	if packetFrameDelta > 0 {
		ratio = packetFrameKey / packetFrameDelta
	}

	// 使用类型安全的max
	if boostRateKey > ratio {
		return boostRateKey
	}
	return ratio
}

// ConvertFECRate 转换FEC率格式
func (p *VCMProtectionMethod) ConvertFECRate(codeRateRTP uint8) uint8 {
	if codeRateRTP >= 255 {
		return 255
	}

	result := uint8(0.5 + 255.0*float64(codeRateRTP)/float64(255-codeRateRTP))
	if result > 255 {
		return 255
	}
	return result
}

// BitsPerFrame 计算每帧比特数
func (p *VCMProtectionMethod) BitsPerFrame(params *VCMProtectionParameters) int {
	// 当时间层可用时，FEC将只应用于基础层
	var bitRateRatio float32 = 1.0
	// 在C++代码中，有一个GetTemporalRateAllocation函数
	// 这里我们简化处理
	if params.numLayers > 1 {
		// 多层编码时的简化计算
		bitRateRatio = 1.0 / float32(params.numLayers)
	}

	frameRateRatio := float32(math.Pow(0.5, float64(params.numLayers-1)))
	bitRate := params.bitRate * bitRateRatio
	frameRate := params.frameRate * frameRateRatio

	// 调整因子
	adjustmentFactor := float32(1.0)

	if frameRate < 1.0 {
		frameRate = 1.0
	}

	// 每帧平均比特数(kbits单位)
	return int(adjustmentFactor * bitRate / frameRate)
}

// UpdateParameters 更新参数并计算保护因子
func (p *VCMProtectionMethod) UpdateParameters(params *VCMProtectionParameters) bool {
	// 首先检查比特率是否太低
	if p.BitRateTooLowForFec(params) {
		p.mu.Lock()
		p.protectionFactorK = 0
		p.protectionFactorD = 0
		p.mu.Unlock()
		return true
	}

	// 计算保护因子
	p.ProtectionFactor(params)

	// 计算有效丢包率
	p.EffectivePacketLoss(params)

	// 保护/fec率是相对于数据包总数定义的
	// RTP模块中的FEC假定保护因子是相对于源数据包数量定义的
	// 所以我们应该转换因子以减少mediaOpt建议率与实际率之间的不匹配
	p.mu.Lock()
	p.protectionFactorK = p.ConvertFECRate(p.protectionFactorK)
	p.protectionFactorD = p.ConvertFECRate(p.protectionFactorD)
	p.mu.Unlock()

	return true
}

// EffectivePacketLoss 计算有效丢包率
func (p *VCMProtectionMethod) EffectivePacketLoss(params *VCMProtectionParameters) bool {
	// 基于RPL(残余丢包率)的编码器有效丢包率
	// 这是基于FEC保护程度的软设置
	// RPL = 接收/输入丢包 - 平均FEC恢复
	// 注意：接收/输入丢包可能基于FilteredLoss进行过滤

	// 当前版本中的有效丢包率，NA
	p.mu.Lock()
	p.effectivePacketLoss = 0
	p.mu.Unlock()
	return true
}

// BitRateTooLowForFec 检查比特率是否太低而不适合使用FEC
func (p *VCMProtectionMethod) BitRateTooLowForFec(params *VCMProtectionParameters) bool {
	// 目前，使用每帧字节的阈值，并考虑帧大小的一些影响
	// 关闭FEC的条件也基于其他因素，如_numLayers，_maxFramesFec和_rtt
	estimateBytesPerFrame := 1000 * p.BitsPerFrame(params) / 8
	maxBytesPerFrame := maxBytesPerFrameForFec
	numPixels := params.codecWidth * params.codecHeight

	if numPixels <= 352*288 {
		maxBytesPerFrame = maxBytesPerFrameForFecLow
	} else if numPixels > 640*480 {
		maxBytesPerFrame = maxBytesPerFrameForFecHigh
	}

	// 最大往返时间阈值(ms)
	const kMaxRttTurnOffFec = 200

	if estimateBytesPerFrame < maxBytesPerFrame &&
		params.numLayers < 3 && params.rtt < kMaxRttTurnOffFec {
		return true
	}

	return false
}

// 辅助函数 - 类型安全的最小值函数
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 辅助函数 - 类型安全的最大值函数
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// MovingAverage 实现递归移动平均
type MovingAverage struct {
	decay       float32
	value       float32
	initialized bool
	mu          sync.Mutex
}

// NewMovingAverage 创建新的移动平均
func NewMovingAverage(decay float32) *MovingAverage {
	return &MovingAverage{
		decay: decay,
	}
}

// Apply 应用移动平均
func (m *MovingAverage) Apply(deltaTimeMs float32, newValue float32) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.initialized {
		m.value = newValue
		m.initialized = true
		return
	}

	// 标准指数移动平均
	alpha := 1.0 - float32(math.Pow(float64(m.decay), float64(deltaTimeMs/1000.0)))
	m.value = alpha*newValue + (1.0-alpha)*m.value
}

// Reset 重置移动平均器
func (m *MovingAverage) Reset(value float32) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = value
	m.initialized = true
}

// Filtered 获取过滤后的值
func (m *MovingAverage) Filtered() float32 {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.value
}
