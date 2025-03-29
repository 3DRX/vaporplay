package gamecapture

import (
	"fmt"
	"image"
	"log/slog"
	"os/exec"
	"time"

	"github.com/3DRX/piongs/config"
	"github.com/pion/mediadevices/pkg/driver"
	"github.com/pion/mediadevices/pkg/frame"
	"github.com/pion/mediadevices/pkg/io/video"
	"github.com/pion/mediadevices/pkg/prop"
)

type screen struct {
	name   string
	reader *nvfbcReader
	tick   *time.Ticker
}

const (
	STEAM_CMD = "steam"
	STEAM_URL = "steam://rungameid/%s"
)

func deviceID(name string) string {
	return fmt.Sprintf("X11Screen_%s", name)
}

// Start the game and block until the game window appears
func Initialize(gameCfg *config.GameConfig) string {
	if gameCfg.GameId != "000000" {
		cmd := exec.Command(STEAM_CMD, fmt.Sprintf(STEAM_URL, gameCfg.GameId))
		_, err := cmd.Output()
		if err != nil {
			panic(err)
		}
	} else {
		slog.Info("no game id specified, skipping game start")
	}
	start := time.Now()
	for {
		// wait until the game window appears, timeout by 30 seconds
		wm, err := openWindow(gameCfg.GameWindowName)
		if err != nil || wm == nil {
			now := time.Now()
			if now.Sub(start) > 60*time.Second {
				panic("failed to find game window")
			}
			slog.Info("waiting for game window", "windowname", gameCfg.GameWindowName)
			time.Sleep(1 * time.Second)
			continue
		}
		defer wm.Close()
		img, err := getShmImageFromWindowMatch(wm)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		// some game have a small loading window, skip it
		if img.img.height < 720 {
			time.Sleep(1 * time.Second)
			continue
		}
		slog.Info("found game window", "windowname", gameCfg.GameWindowName)
		break
	}
	slog.Info("initializing game capture", "windowname", gameCfg.GameWindowName)
	labelName := deviceID(gameCfg.GameWindowName)
	driver.GetManager().Register(
		&screen{
			name: gameCfg.GameWindowName,
		},
		driver.Info{
			Label:      labelName,
			DeviceType: driver.Camera,
		},
	)
	return labelName
}

func (s *screen) Open() error {
	r, err := newNvFBCReader()
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

	reader := s.reader

	r := video.ReaderFunc(func() (image.Image, func(), error) {
		<-s.tick.C
		img := reader.Read()
		return img, func() {}, nil
	})
	return r, nil
}

func (s *screen) Properties() []prop.Media {
	w, h := s.reader.Size()
	slog.Info("game capture properties", "width", w, "height", h)
	return []prop.Media{
		{
			DeviceID: deviceID(s.name),
			Video: prop.Video{
				Width:       w,
				Height:      h,
				FrameFormat: frame.FormatRGBA,
			},
		},
	}
}
