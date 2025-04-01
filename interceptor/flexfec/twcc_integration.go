package flexfec

import (
	"sync"
	"time"
)

// TWCCStats 收集 TWCC 统计信息
type TWCCStats struct {
	mu         sync.Mutex
	lossRate   float32
	rtt        int64
	bitRate    int
	lastUpdate time.Time
}

// NewTWCCStats 创建 TWCC 统计收集器
func NewTWCCStats() *TWCCStats {
	return &TWCCStats{
		lastUpdate: time.Now(),
	}
}

// UpdateStats 更新网络统计数据
func (t *TWCCStats) UpdateStats(lossRate float32, rtt int64, bitRate int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.lossRate = lossRate
	t.rtt = rtt
	t.bitRate = bitRate
	t.lastUpdate = time.Now()
}

// GetStats 获取当前统计数据
func (t *TWCCStats) GetStats() (float32, int64, int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.lossRate, t.rtt, t.bitRate
}

// SetupTWCCFeedback 设置 TWCC 反馈到 FEC 拦截器
func SetupTWCCFeedback(fecInterceptor *FecInterceptor, twccStats *TWCCStats) {
	// 启动定时器定期更新 FEC 参数
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for range ticker.C {
			lossRate, rtt, _ := twccStats.GetStats()

			// 更新 FEC 拦截器的网络参数
			fecInterceptor.SetLossRate(lossRate)
			fecInterceptor.SetRTT(rtt)
		}
	}()
}
