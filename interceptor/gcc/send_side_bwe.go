// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package gcc

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"sync"
	"time"

	"github.com/3DRX/piongs/interceptor/cc"
	"github.com/3DRX/piongs/interceptor/ntp"
	"github.com/pion/interceptor"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

const (
	transportCCURI = "http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01"
	latestBitrate  = 10_000
	minBitrate     = 5_000
	maxBitrate     = 50_000_000
)

// ErrSendSideBWEClosed is raised when SendSideBWE.WriteRTCP is called after SendSideBWE.Close.
var ErrSendSideBWEClosed = errors.New("SendSideBwe closed")

// Pacer is the interface implemented by packet pacers.
type Pacer interface {
	interceptor.RTPWriter
	AddStream(ssrc uint32, writer interceptor.RTPWriter)
	SetTargetBitrate(int)
	Close() error
}

// Stats contains internal statistics of the bandwidth estimator.
type Stats struct {
	LossStats
	DelayStats
}

type CCStats struct {
	rtt  time.Duration
	acks []cc.Acknowledgment
	twcc *rtcp.TransportLayerCC
}

// SendSideBWE implements a combination of loss and delay based GCC.
type SendSideBWE struct {
	pacer           Pacer
	lossController  *lossBasedBandwidthEstimator
	delayController *delayController
	feedbackAdapter *cc.FeedbackAdapter

	onTargetBitrateChange func(bitrate int)

	lock          sync.Mutex
	latestStats   Stats
	latestBitrate int
	minBitrate    int
	maxBitrate    int

	close     chan struct{}
	closeLock sync.RWMutex

	statsChan       chan CCStats
	rfc8888Chan     chan []cc.Acknowledgment
	latestStatsChan chan Stats
}

// Option configures a bandwidth estimator.
type Option func(*SendSideBWE) error

// SendSideBWEInitialBitrate sets the initial bitrate of new GCC interceptors.
func SendSideBWEInitialBitrate(rate int) Option {
	return func(e *SendSideBWE) error {
		e.latestBitrate = rate

		return nil
	}
}

// SendSideBWEMaxBitrate sets the initial bitrate of new GCC interceptors.
func SendSideBWEMaxBitrate(rate int) Option {
	return func(e *SendSideBWE) error {
		e.maxBitrate = rate

		return nil
	}
}

// SendSideBWEMinBitrate sets the initial bitrate of new GCC interceptors.
func SendSideBWEMinBitrate(rate int) Option {
	return func(e *SendSideBWE) error {
		e.minBitrate = rate

		return nil
	}
}

// SendSideBWEPacer sets the pacing algorithm to use.
func SendSideBWEPacer(p Pacer) Option {
	return func(e *SendSideBWE) error {
		e.pacer = p

		return nil
	}
}

// NewSendSideBWE creates a new sender side bandwidth estimator.
func NewSendSideBWE(opts ...Option) (*SendSideBWE, error) {
	statsChan := make(chan CCStats, 100)
	rfc8888Chan := make(chan []cc.Acknowledgment, 100)
	latestStatsChan := make(chan Stats, 100)
	send := &SendSideBWE{
		pacer:                 nil,
		lossController:        nil,
		delayController:       nil,
		feedbackAdapter:       cc.NewFeedbackAdapter(),
		onTargetBitrateChange: nil,
		lock:                  sync.Mutex{},
		latestStats:           Stats{},
		latestBitrate:         latestBitrate,
		minBitrate:            minBitrate,
		maxBitrate:            maxBitrate,
		close:                 make(chan struct{}),
		statsChan:             statsChan,
		rfc8888Chan:           rfc8888Chan,
		latestStatsChan:       latestStatsChan,
	}
	for _, opt := range opts {
		if err := opt(send); err != nil {
			return nil, err
		}
	}
	if send.pacer == nil {
		send.pacer = NewLeakyBucketPacer(send.latestBitrate)
	}
	send.lossController = newLossBasedBWE(send.latestBitrate)
	send.delayController = newDelayController(delayControllerConfig{
		nowFn:          time.Now,
		initialBitrate: send.latestBitrate,
		minBitrate:     send.minBitrate,
		maxBitrate:     send.maxBitrate,
	})

	send.delayController.onUpdate(send.onDelayUpdate)

	go StatsThread(statsChan, rfc8888Chan, latestStatsChan)

	return send, nil
}

// AddStream adds a new stream to the bandwidth estimator.
func (e *SendSideBWE) AddStream(info *interceptor.StreamInfo, writer interceptor.RTPWriter) interceptor.RTPWriter {
	var hdrExtID uint8
	for _, e := range info.RTPHeaderExtensions {
		if e.URI == transportCCURI {
			hdrExtID = uint8(e.ID)
			break
		}
	}

	e.pacer.AddStream(info.SSRC, interceptor.RTPWriterFunc(
		func(header *rtp.Header, payload []byte, attributes interceptor.Attributes) (int, error) {
			if hdrExtID != 0 {
				if attributes == nil {
					attributes = make(interceptor.Attributes)
				}
				attributes.Set(cc.TwccExtensionAttributesKey, hdrExtID)
			}
			if err := e.feedbackAdapter.OnSent(time.Now(), header, len(payload), attributes); err != nil {
				return 0, err
			}

			return writer.Write(header, payload, attributes)
		},
	))

	return e.pacer
}

// WriteRTCP adds some RTCP feedback to the bandwidth estimator.
//
//nolint:cyclop
func (e *SendSideBWE) WriteRTCP(pkts []rtcp.Packet, _ interceptor.Attributes) error {
	now := time.Now()
	e.closeLock.RLock()
	defer e.closeLock.RUnlock()

	if e.isClosed() {
		return ErrSendSideBWEClosed
	}

	for _, pkt := range pkts {
		var acks []cc.Acknowledgment
		var err error
		var feedbackSentTime time.Time
		switch fb := pkt.(type) {
		case *rtcp.TransportLayerCC:
			acks, err = e.feedbackAdapter.OnTransportCCFeedback(now, fb)
			if err != nil {
				return err
			}
			for i, ack := range acks {
				if i == 0 {
					feedbackSentTime = ack.Arrival

					continue
				}
				if ack.Arrival.After(feedbackSentTime) {
					feedbackSentTime = ack.Arrival
				}
			}

			feedbackMinRTT := time.Duration(math.MaxInt)
			for _, ack := range acks {
				if ack.Arrival.IsZero() {
					continue
				}
				pendingTime := feedbackSentTime.Sub(ack.Arrival)
				rtt := now.Sub(ack.Departure) - pendingTime
				feedbackMinRTT = time.Duration(minInt(int(rtt), int(feedbackMinRTT)))
			}
			if feedbackMinRTT < math.MaxInt {
				e.delayController.updateRTT(feedbackMinRTT)
			}

			e.statsChan <- CCStats{
				acks: acks,
				rtt:  feedbackMinRTT,
				twcc: fb,
			}

			e.lossController.updateLossEstimate(acks)
			e.delayController.updateDelayEstimate(acks)
		case *rtcp.CCFeedbackReport:
			acks = e.feedbackAdapter.OnRFC8888Feedback(now, fb)
			feedbackSentTime = ntp.ToTime(uint64(fb.ReportTimestamp) << 16)
			e.rfc8888Chan <- acks
		default:
			continue
		}

	}

	return nil
}

// GetTargetBitrate returns the current target bitrate in bits per second.
func (e *SendSideBWE) GetTargetBitrate() int {
	e.lock.Lock()
	defer e.lock.Unlock()

	return e.latestBitrate
}

// GetStats returns some internal statistics of the bandwidth estimator.
func (e *SendSideBWE) GetStats() map[string]interface{} {
	e.lock.Lock()
	defer e.lock.Unlock()

	return map[string]interface{}{
		"lossTargetBitrate":  e.latestStats.LossStats.TargetBitrate,
		"averageLoss":        e.latestStats.AverageLoss,
		"delayTargetBitrate": e.latestStats.DelayStats.TargetBitrate,
		"delayMeasurement":   float64(e.latestStats.Measurement.Microseconds()) / 1000.0,
		"delayEstimate":      float64(e.latestStats.Estimate.Microseconds()) / 1000.0,
		"delayThreshold":     float64(e.latestStats.Threshold.Microseconds()) / 1000.0,
		"usage":              e.latestStats.Usage.String(),
		"state":              e.latestStats.State.String(),
	}
}

// OnTargetBitrateChange sets the callback that is called when the target
// bitrate in bits per second changes.
func (e *SendSideBWE) OnTargetBitrateChange(f func(bitrate int)) {
	e.onTargetBitrateChange = f
}

// isClosed returns true if SendSideBWE is closed.
func (e *SendSideBWE) isClosed() bool {
	select {
	case <-e.close:
		return true
	default:
		return false
	}
}

// Close stops and closes the bandwidth estimator.
func (e *SendSideBWE) Close() error {
	e.closeLock.Lock()
	defer e.closeLock.Unlock()

	if err := e.delayController.Close(); err != nil {
		return err
	}
	close(e.close)

	return e.pacer.Close()
}

func (e *SendSideBWE) onDelayUpdate(delayStats DelayStats) {
	e.lock.Lock()
	defer e.lock.Unlock()

	lossStats := e.lossController.getEstimate(delayStats.TargetBitrate)
	bitrate := minInt(delayStats.TargetBitrate, lossStats.TargetBitrate)
	if bitrate != e.latestBitrate {
		e.latestBitrate = bitrate
		e.pacer.SetTargetBitrate(e.latestBitrate)
	}

	if e.onTargetBitrateChange != nil {
		go e.onTargetBitrateChange(bitrate)
	}

	e.latestStats = Stats{
		LossStats:  lossStats,
		DelayStats: delayStats,
	}
	e.latestStatsChan <- e.latestStats
}

////// stats only

func StatsThread(statsChan chan CCStats, rfc8888Chan chan []cc.Acknowledgment, gccStatsChan chan Stats) {
	f, err := os.Create("gcc_stats.csv")
	if err != nil {
		panic(err)
	}
	w := bufio.NewWriter(f)
	w.WriteString("twcc_id,frame_size,loss_packets_counts,threshold,delay_grad_before_kalman,delay_grad_after_kalman,gcc_bw,rtt\n")
	defer f.Close()
	f2, err := os.Create("rfc8888.csv")
	if err != nil {
		panic(err)
	}
	w2 := bufio.NewWriter(f2)
	w2.WriteString("sequence_number,ssrc,size,departure,arrival,ecn\n")
	defer f2.Close()
	index := 0
	index2 := 0
	for {
		select {
		case statsItem := <-statsChan:
			acks := statsItem.acks
			rtt := statsItem.rtt
			gccStats := <-gccStatsChan
			delayGradBeforeKalman := gccStats.DelayStats.Measurement.Microseconds()
			delayGradAfterKalman := gccStats.DelayStats.Estimate.Microseconds()
			threshold := gccStats.DelayStats.Threshold.Microseconds()
			gccBw := minInt(gccStats.DelayStats.TargetBitrate, gccStats.LossStats.TargetBitrate)
			for _, frame := range groupByFrame(acks) {
				frameSize := len(frame)
				lossPacketsCount := getLossPacketsCounts(frame)
				_, err := w.WriteString(
					fmt.Sprintf(
						"%d,%d,%d,%d,%d,%d,%d,%d\n",
						index,
						frameSize,
						lossPacketsCount,
						threshold,
						delayGradBeforeKalman,
						delayGradAfterKalman,
						gccBw,
						rtt.Milliseconds(),
					),
				)
				if err != nil {
					slog.Error("failed to transport packets feedback to file", "error", err)
				}
			}
			if index%20 == 0 {
				w.Flush()
			}
			index++
		case statsItem := <-rfc8888Chan:
			for _, p := range statsItem {
				_, err := w2.WriteString(
					fmt.Sprintf(
						"%d,%d,%d,%d,%d,%d\n",
						p.SequenceNumber,
						p.SSRC,
						p.Size,
						int64(float64(p.Departure.UnixMilli())),
						int64(float64(p.Arrival.UnixMilli())),
						p.ECN,
					),
				)
				if err != nil {
					slog.Error("failed to transport packets feedback to file", "error", err)
				}
			}
			if index2%270 == 0 {
				w2.Flush()
			}
			index2++
		}
	}
}

func groupByFrame(acks []cc.Acknowledgment) [][]cc.Acknowledgment {
	groupedMap := make(map[uint16][]cc.Acknowledgment)
	for _, ack := range acks {
		ack := ack
		groupedMap[ack.Stats.FrameID] = append(groupedMap[ack.Stats.FrameID], ack)
	}
	grouped := make([][]cc.Acknowledgment, 0, len(groupedMap))
	// ensure the order of the frames
	for i := acks[0].Stats.FrameID; i < acks[len(acks)-1].Stats.FrameID; i++ {
		frame, ok := groupedMap[i]
		if ok {
			grouped = append(grouped, frame)
		}
	}
	return grouped
}

func getLossPacketsCounts(frame []cc.Acknowledgment) int {
	count := 0
	for _, ack := range frame {
		if ack.Arrival.IsZero() {
			count++
		}
	}
	return count
}
