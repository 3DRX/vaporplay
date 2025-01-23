package gamecapture

import (
	"fmt"
	"image"
	"log/slog"
	"time"

	"github.com/pion/mediadevices/pkg/driver"
	"github.com/pion/mediadevices/pkg/frame"
	"github.com/pion/mediadevices/pkg/io/video"
	"github.com/pion/mediadevices/pkg/prop"
)

type screen struct {
	num    int
	name string
	reader *reader
	tick   *time.Ticker
}

func deviceID(num int) string {
	return fmt.Sprintf("X11Screen%d", num)
}

func init() {
	Initialize("Firewatch")
}

// Initialize finds and registers active displays. This is part of an experimental API.
func Initialize(windowname string) {
	slog.Info("Registering screen", "numScreen", 0)
	driver.GetManager().Register(
		&screen{
			num: 0,
			name: windowname,
		},
		driver.Info{
			Label:      deviceID(0),
			DeviceType: driver.Camera,
		},
	)
}

func (s *screen) Open() error {
	r, err := newReader(s.name)
	if err != nil {
		return err
	}
	s.reader = r
	return nil
}

func (s *screen) Close() error {
	s.reader.Close()
	if s.tick != nil {
		s.tick.Stop()
	}
	return nil
}

func (s *screen) VideoRecord(p prop.Media) (video.Reader, error) {
	if p.FrameRate == 0 {
		p.FrameRate = 10
	}
	s.tick = time.NewTicker(time.Duration(float32(time.Second) / p.FrameRate))

	var dst image.RGBA
	reader := s.reader

	r := video.ReaderFunc(func() (image.Image, func(), error) {
		<-s.tick.C
		return reader.Read().ToRGBA(&dst), func() {}, nil
	})
	return r, nil
}

func (s *screen) Properties() []prop.Media {
	rect := s.reader.img.Bounds()
	w := rect.Dx()
	h := rect.Dy()
	return []prop.Media{
		{
			DeviceID: deviceID(s.num),
			Video: prop.Video{
				Width:       w,
				Height:      h,
				FrameFormat: frame.FormatRGBA,
			},
		},
	}
}
