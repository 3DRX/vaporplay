// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package gcc

import (
	"bufio"
	"container/list"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/pion/interceptor"
	"github.com/pion/logging"
	"github.com/pion/rtp"
)

var errLeakyBucketPacerPoolCastFailed = errors.New("failed to access leaky bucket pacer pool, cast failed")

type item struct {
	header     *rtp.Header
	payload    *[]byte
	size       int
	attributes interceptor.Attributes
}

type StatsItem struct {
	budget        int
	egress        int
	egressCount   int
	ingress       int
	ingressCount  int
	targetBitrate int
	bufferCount   int
}

// LeakyBucketPacer implements a leaky bucket pacing algorithm.
type LeakyBucketPacer struct {
	log logging.LeveledLogger

	f                 float64
	targetBitrate     int
	targetBitrateLock sync.Mutex

	pacingInterval time.Duration

	qLock sync.RWMutex
	queue *list.List
	done  chan struct{}

	ssrcToWriter map[uint32]interceptor.RTPWriter
	writerLock   sync.RWMutex

	// for stats tracing
	statsChan    chan StatsItem
	iLock        sync.RWMutex
	ingress      int
	ingressCount int

	pool *sync.Pool
}

// NewLeakyBucketPacer initializes a new LeakyBucketPacer.
func NewLeakyBucketPacer(initialBitrate int) *LeakyBucketPacer {
	pacer := &LeakyBucketPacer{
		log:            logging.NewDefaultLoggerFactory().NewLogger("pacer"),
		f:              1.5,
		targetBitrate:  initialBitrate,
		pacingInterval: 5 * time.Millisecond,
		qLock:          sync.RWMutex{},
		queue:          list.New(),
		done:           make(chan struct{}),
		ssrcToWriter:   map[uint32]interceptor.RTPWriter{},
		pool:           &sync.Pool{},
		statsChan:      make(chan StatsItem, 10),
	}
	pacer.pool = &sync.Pool{
		New: func() interface{} {
			b := make([]byte, 1460)

			return &b
		},
	}

	go pacer.Run()

	go PacerStatsThread(pacer.statsChan)

	return pacer
}

// AddStream adds a new stream and its corresponding writer to the pacer.
func (p *LeakyBucketPacer) AddStream(ssrc uint32, writer interceptor.RTPWriter) {
	p.writerLock.Lock()
	defer p.writerLock.Unlock()
	p.ssrcToWriter[ssrc] = writer
}

// SetTargetBitrate updates the target bitrate at which the pacer is allowed to
// send packets. The pacer may exceed this limit by p.f.
func (p *LeakyBucketPacer) SetTargetBitrate(rate int) {
	p.targetBitrateLock.Lock()
	defer p.targetBitrateLock.Unlock()
	p.targetBitrate = int(p.f * float64(rate))
}

func (p *LeakyBucketPacer) getTargetBitrate() int {
	p.targetBitrateLock.Lock()
	defer p.targetBitrateLock.Unlock()

	return p.targetBitrate
}

// Write sends a packet with header and payload the a previously registered
// stream.
func (p *LeakyBucketPacer) Write(header *rtp.Header, payload []byte, attributes interceptor.Attributes) (int, error) {
	buf, ok := p.pool.Get().(*[]byte)
	if !ok {
		return 0, errLeakyBucketPacerPoolCastFailed
	}

	copy(*buf, payload)
	hdr := header.Clone()
	// fmt.Printf(
	// 	"new rtp packet arrived at pacer, seq=%d, timestamp=%d, marker=%v\n",
	// 	hdr.SequenceNumber,
	// 	hdr.Timestamp,
	// 	hdr.Marker,
	// )

	p.qLock.Lock()
	p.queue.PushBack(&item{
		header:     &hdr,
		payload:    buf,
		size:       len(payload),
		attributes: attributes,
	})
	p.qLock.Unlock()

	n := header.MarshalSize() + len(payload)
	p.iLock.Lock()
	p.ingress += n
	p.ingressCount += 1
	p.iLock.Unlock()

	return n, nil
}

// Run starts the LeakyBucketPacer.
func (p *LeakyBucketPacer) Run() {
	ticker := time.NewTicker(p.pacingInterval)
	defer ticker.Stop()

	start := false

	lastSent := time.Now()
	for {
		select {
		case <-p.done:
			return
		case now := <-ticker.C:
			budget := int(float64(now.Sub(lastSent).Milliseconds()) * float64(p.getTargetBitrate()) / 8000.0)
			fullBudget := budget
			writeSuccess := true
			egressCount := 0
			bufferCount := 0
			p.qLock.Lock()
			for p.queue.Len() != 0 && budget > 0 {
				p.log.Infof("budget=%v, len(queue)=%v, targetBitrate=%v", budget, p.queue.Len(), p.getTargetBitrate())
				next, ok := p.queue.Remove(p.queue.Front()).(*item)
				p.qLock.Unlock()
				if !ok {
					p.log.Warnf("failed to access leaky bucket pacer queue, cast failed")

					continue
				}

				p.writerLock.RLock()
				writer, ok := p.ssrcToWriter[next.header.SSRC]
				p.writerLock.RUnlock()
				if !ok {
					p.log.Warnf("no writer found for ssrc: %v", next.header.SSRC)
					p.pool.Put(next.payload)
					p.qLock.Lock()

					continue
				}

				n, err := writer.Write(next.header, (*next.payload)[:next.size], next.attributes)
				if err != nil {
					writeSuccess = false
					p.log.Errorf("failed to write packet: %v", err)
				}
				lastSent = now
				budget -= n
				egressCount += 1

				p.pool.Put(next.payload)
				p.qLock.Lock()
			}
			bufferCount = p.queue.Len()
			p.qLock.Unlock()

			if writeSuccess && budget != fullBudget {
				start = true
			}

			if start {
				p.iLock.Lock()
				ingress := p.ingress
				ingressCount := p.ingressCount
				p.ingress = 0
				p.ingressCount = 0
				p.iLock.Unlock()
				statsItem := StatsItem{
					budget:        fullBudget,
					egress:        fullBudget - budget,
					egressCount:   egressCount,
					ingress:       ingress,
					ingressCount:  ingressCount,
					targetBitrate: p.getTargetBitrate(),
					bufferCount:   bufferCount,
				}
				p.statsChan <- statsItem
			}
		}
	}
}

// Close closes the LeakyBucketPacer.
func (p *LeakyBucketPacer) Close() error {
	close(p.done)

	return nil
}

func PacerStatsThread(statsChan chan StatsItem) {
	// open file for writing
	f, err := os.Create("leaky_bucket_pacer.csv")
	if err != nil {
		panic(err)
	}
	w := bufio.NewWriter(f)
	w.WriteString("budget,egress,egress_count,ingress,ingress_count,target_bitrate,buffer_count\n")
	defer f.Close()
	index := 0
	var statsItem StatsItem
	for {
		select {
		case statsItem = <-statsChan:
			_, err := w.WriteString(fmt.Sprintf(
				"%d,%d,%d,%d,%d,%d,%d\n",
				statsItem.budget,
				statsItem.egress,
				statsItem.egressCount,
				statsItem.ingress,
				statsItem.ingressCount,
				statsItem.targetBitrate,
				statsItem.bufferCount,
			))
			if err != nil {
				slog.Error("failed to write pacer stats to file", "error", err)
			}
			if index%200 == 0 {
				w.Flush()
			}
			index++
		}
	}
}
