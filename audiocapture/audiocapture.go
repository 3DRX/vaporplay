// Package audiolin provides Linux audio driver using PulseAudio.
package audiocapture

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/jfreymuth/pulse"
	"github.com/pion/mediadevices/pkg/driver"
	"github.com/pion/mediadevices/pkg/io/audio"
	"github.com/pion/mediadevices/pkg/prop"
	"github.com/pion/mediadevices/pkg/wave"
)

func init() {
	driver.GetManager().Register(
		&pulseDriver{}, driver.Info{Label: "PulseAudio", DeviceType: driver.Microphone},
	)
}

type pulseDriver struct {
	client     *pulse.Client
	stream     *pulse.RecordStream
	closed     <-chan struct{}
	cancel     func()
	sampleRate int
	channels   int
}

func (d *pulseDriver) Open() error {
	var err error
	d.client, err = pulse.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create PulseAudio client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	d.closed = ctx.Done()
	d.cancel = cancel
	return nil
}

func (d *pulseDriver) Close() error {
	if d.stream != nil {
		d.stream.Close()
	}
	if d.client != nil {
		d.client.Close()
	}
	d.cancel()
	return nil
}

func (d *pulseDriver) AudioRecord(p prop.Media) (audio.Reader, error) {
	var err error
	d.sampleRate = p.SampleRate
	d.channels = p.ChannelCount

	// Configure PulseAudio stream
	d.stream, err = d.client.NewRecord(
		pulse.RecordSettings{
			Format:   pulse.Float32LE,
			Channels: p.ChannelCount,
			Rate:     p.SampleRate,
			Latency:  p.Latency,
		},
		// Buffer size calculation based on latency
		int(float64(p.SampleRate)*p.Latency.Seconds())*p.ChannelCount*4,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create record stream: %v", err)
	}

	closed := d.closed
	nSamples := int(uint64(p.SampleRate) * uint64(p.Latency) / uint64(time.Second))

	reader := audio.ReaderFunc(func() (wave.Audio, func(), error) {
		select {
		case <-closed:
			return nil, func() {}, io.EOF
		default:
		}

		// Create audio buffer
		a := wave.NewFloat32Interleaved(
			wave.ChunkInfo{
				Channels:     p.ChannelCount,
				Len:         nSamples,
				SamplingRate: p.SampleRate,
			},
		)

		// Read from PulseAudio stream
		buffer := make([]float32, nSamples*p.ChannelCount)
		n, err := d.stream.Read(buffer)
		if err != nil {
			return nil, func() {}, err
		}
		if n == 0 {
			return nil, func() {}, io.EOF
		}

		// Copy data to wave.Audio buffer
		for i := 0; i < n/p.ChannelCount; i++ {
			for ch := 0; ch < p.ChannelCount; ch++ {
				a.SetFloat32(i, ch, wave.Float32Sample(buffer[i*p.ChannelCount+ch]))
			}
		}

		return a, func() {}, nil
	})

	return reader, nil
}

func (d *pulseDriver) Properties() []prop.Media {
	return []prop.Media{
		{
			Audio: prop.Audio{
				SampleRate:   44100,
				Latency:      time.Millisecond * 20,
				ChannelCount: 1,
			},
		},
		{
			Audio: prop.Audio{
				SampleRate:   44100,
				Latency:      time.Millisecond * 20,
				ChannelCount: 2,
			},
		},
		{
			Audio: prop.Audio{
				SampleRate:   48000,
				Latency:      time.Millisecond * 20,
				ChannelCount: 1,
			},
		},
		{
			Audio: prop.Audio{
				SampleRate:   48000,
				Latency:      time.Millisecond * 20,
				ChannelCount: 2,
			},
		},
	}
}
