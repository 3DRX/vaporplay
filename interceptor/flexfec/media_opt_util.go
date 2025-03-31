package flexfec

import (
    "math"
    "sync"
    "time"
)

// ProtectionParams 存储 FEC 保护参数
type ProtectionParams struct {
    // 保护因子 (0-255)
    ProtectionFactorDelta uint8  // P帧保护因子
    ProtectionFactorKey   uint8  // I帧保护因子
    
	fec_rate uint8

    // 网络与媒体参数
    RTT               time.Duration
    LossRate          float32
    AvailableBitrate  int
    FrameRate         float32
	PacketsPerFrame   float32 // 每帧的平均包数
    PacketsPerFrameKey float32 // 每个关键帧的平均包数
    CodecWidth        int
    CodecHeight       int
	NumLayers         int     //SVC 层数，暂未使用
}

// ProtectionCalculator 计算 FEC 保护参数
type ProtectionCalculator struct {
    mu          sync.Mutex
    params      ProtectionParams
    statsProvider *StatsProvider
    
    // 重置标志
    paramsDirty bool
}

// NewProtectionCalculator 创建保护计算器
func NewProtectionCalculator(statsProvider *StatsProvider) *ProtectionCalculator {
    return &ProtectionCalculator{
        statsProvider: statsProvider,
        params: ProtectionParams{
            FrameRate: 30,
            CodecWidth: 640,
            CodecHeight: 480,
        },
        paramsDirty: true,
    }
}

// UpdateCodecParams 更新编码参数
func (p *ProtectionCalculator) UpdateCodecParams(width, height int, frameRate float32) {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    p.params.CodecWidth = width
    p.params.CodecHeight = height
    p.params.FrameRate = frameRate
    p.paramsDirty = true
}

// GetProtectionParams 返回计算的保护参数
func (p *ProtectionCalculator) GetProtectionParams() ProtectionParams {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    if p.paramsDirty {
        p.updateProtectionFactors()
        p.paramsDirty = false
    }
    
    return p.params
}

// CalculateFecPackets 计算应该生成的 FEC 包数量
func (p *ProtectionCalculator) CalculateFecPackets(mediaPacketCount int, isKeyFrame bool) uint32 {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    if p.paramsDirty {
        p.updateProtectionFactors()
        p.paramsDirty = false
    }
    
    // 选择适当的保护因子
    var protectionFactor uint8
    if isKeyFrame {
        protectionFactor = p.params.ProtectionFactorKey
    } else {
        protectionFactor = p.params.ProtectionFactorDelta
    }
    
    // 转换保护因子到 FEC 包数量
    return p.protectionFactorToFecCount(protectionFactor, mediaPacketCount)
}

// 核心方法：更新保护因子
func (p *ProtectionCalculator) updateProtectionFactors() {
    // 获取网络状态
    stats := p.statsProvider.GetStats()
    
    // 更新内部参数
    p.params.RTT = stats.RTT
    p.params.LossRate = stats.LossRate
    p.params.AvailableBitrate = stats.AvailableBitrate
    
    // 计算基础保护因子
    // 1. 计算有效比特率
    bitsPerFrame := p.calculateBitsPerFrame()
    
    // 2. 调整因子
    spatialFactor := p.calculateSpatialFactor()
    effectiveRate := float32(bitsPerFrame) * spatialFactor
    
    // 3. 计算表格索引
    rateIndex := p.calculateRateIndex(effectiveRate)
    lossIndex := p.calculateLossIndex(p.params.LossRate)
    
    // 4. 查表获取基础保护因子
    var deltaFactor uint8
    if rateIndex < len(FecRateTable) && lossIndex < PacketLossMax {
        if len(FecRateTable[rateIndex]) > int(lossIndex) {
            deltaFactor = FecRateTable[rateIndex][lossIndex]
        }
    }
    
    // 5. 计算关键帧保护因子 (通常为P帧的2-3倍)
    keyFactor := p.calculateKeyFrameFactor(deltaFactor)
    
    // 6. 转换为源包相对的保护因子 (WebRTC 使用的格式)
    p.params.ProtectionFactorDelta = p.convertFecRate(deltaFactor)
    p.params.ProtectionFactorKey = p.convertFecRate(keyFactor)
}

// 辅助方法

// 计算每帧比特数
func (p *ProtectionCalculator) calculateBitsPerFrame() int {
    return int(float32(p.params.AvailableBitrate) / p.params.FrameRate)
}

// 计算空间调整因子
func (p *ProtectionCalculator) calculateSpatialFactor() float32 {
    // 相对于参考分辨率 (704x576) 的因子
    referencePixels := float32(704 * 576)
    actualPixels := float32(p.params.CodecWidth * p.params.CodecHeight)
    
    // 使用 0.3 的指数可以平滑调整的影响
    return float32(math.Pow(float64(actualPixels/referencePixels), -0.3))
}

// 计算比特率索引
func (p *ProtectionCalculator) calculateRateIndex(effectiveRate float32) int {
    // 将有效比特率映射到 FecRateTable 索引
    // 表格范围通常从 200kbps (~5) 到 8000kbps (~49)
    const ratePar1 = 5.0
    const ratePar2 = 49
    
    rateIndex := int((float64(effectiveRate) - ratePar1) / ratePar1)
    if rateIndex < 0 {
        rateIndex = 0
    }
    if rateIndex > ratePar2 {
        rateIndex = ratePar2
    }
    
    // 确保在表格范围内
    if rateIndex >= len(FecRateTable) {
        rateIndex = len(FecRateTable) - 1
    }
    
    return rateIndex
}

// 计算丢包率索引
func (p *ProtectionCalculator) calculateLossIndex(lossRate float32) uint8 {
    // 将丢包率 (0-1) 转换为索引 (0-128)
    lossIndex := uint8(lossRate * float32(PacketLossMax-1))
    if lossIndex >= PacketLossMax {
        lossIndex = PacketLossMax - 1
    }
    return lossIndex
}

// 计算关键帧保护因子
func (p *ProtectionCalculator) calculateKeyFrameFactor(deltaFactor uint8) uint8 {
    // 关键帧保护通常是P帧的2-3倍
    keyFactor := uint8(math.Min(255, float64(deltaFactor)*2.5))
    
    // RTT 高时增加保护
    if p.params.RTT > 100*time.Millisecond && keyFactor < 30 {
        keyFactor = 30
    }
    
    return keyFactor
}

// 转换 FEC 率格式
func (p *ProtectionCalculator) convertFecRate(codeRate uint8) uint8 {
    // 从总包数相对比例转换为源包数相对比例
    // FEC/(FEC+Media) -> FEC/Media
    if codeRate >= 255 {
        return 255
    }
    
    return uint8(math.Min(255, (0.5 + 255.0*float64(codeRate)/float64(255-codeRate))))
}

// 从保护因子计算 FEC 包数量
func (p *ProtectionCalculator) protectionFactorToFecCount(protectionFactor uint8, mediaPacketCount int) uint32 {
    // 保护因子是相对于源包的比例
    fecRatio := float64(protectionFactor) / 255.0
    
    // 防止比率接近1导致除零问题
    if fecRatio > 0.9 {
        fecRatio = 0.9
    }
    
    fecCount := fecRatio * float64(mediaPacketCount)
    return uint32(math.Round(fecCount))
}