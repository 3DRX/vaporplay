package ui

import (
	"log/slog"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	clientconfig "github.com/3DRX/vaporplay/client/vaporplay-native-client/client-config"
)

type UIThread struct {
	startGamePromise chan *clientconfig.ClientConfig
	game             *ebitenGame
}

type ebitenGame struct {
	frame *ebiten.Image

	lock sync.Mutex
}

func NewUIThread() (*UIThread, chan *clientconfig.ClientConfig) {
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("VaporPlay")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetVsyncEnabled(false)

	game := &ebitenGame{}

	startGamePromise := make(chan *clientconfig.ClientConfig)

	return &UIThread{
		startGamePromise: startGamePromise,
		game:             game,
	}, startGamePromise
}

func (u *UIThread) Spin() {
	imgGenerator := VideoRecord()
	go func() {
		ticker := time.NewTicker(8300 * time.Microsecond)
		for {
			select {
			case <-ticker.C:
				img := imgGenerator()
				u.game.lock.Lock()
				u.game.frame = ebiten.NewImageFromImage(img)
				u.game.lock.Unlock()
			default:
			}
		}
	}()

	err := ebiten.RunGame(u.game)
	if err != nil {
		slog.Error("ebiten error", "error", err)
	}
}

// func (u *UIThread) doSomeThing() {
// 	// wait for 2 seconds, mock user input
// 	time.Sleep(2 * time.Second)

// 	cfg := &clientconfig.ClientConfig{
// 		Addr: "localhost:8080",
// 		SessionConfig: config.SessionConfig{
// 			GameConfig: config.GameConfig{
// 				GameId:          "383870",
// 				GameWindowName:  "Firewatch",
// 				GameDisplayName: "Fire Watch",
// 				GameIcon:        "",
// 				EndGameCommands: []config.KillProcessCommandConfig{{
// 					Flags:       []string{},
// 					ProcessName: "fw.x86_64",
// 				}},
// 			},
// 			CodecConfig: config.CodecConfig{
// 				Codec:          "h264_nvenc",
// 				InitialBitrate: 5_000_000,
// 				MaxBitrate:     20_000_000,
// 				FrameRate:      60,
// 			},
// 		},
// 	}
// 	slog.Info("start game", "game_id", cfg.SessionConfig.GameConfig.GameId)
// 	u.startGamePromise <- cfg
// }

func (g *ebitenGame) Update() error {
	return nil
}

func (g *ebitenGame) Draw(screen *ebiten.Image) {
	g.lock.Lock()
	defer g.lock.Unlock()

	if g.frame == nil {
		return
	}

	var windowWidth, windowHeight int

	if ebiten.IsFullscreen() {
		windowWidth, windowHeight = ebiten.Monitor().Size()
	} else {
		windowWidth, windowHeight = ebiten.WindowSize()
	}
	s := ebiten.Monitor().DeviceScaleFactor()
	windowWidth = int(float64(windowWidth) * s)
	windowHeight = int(float64(windowHeight) * s)
	w := g.frame.Bounds().Dx()
	h := g.frame.Bounds().Dy()
	scaleX := float64(windowWidth) / float64(w)
	scaleY := float64(windowHeight) / float64(h)
	scale := min(scaleY, scaleX)

	scaledWidth := float64(w) * scale
	scaledHeight := float64(h) * scale
	offsetX := (float64(windowWidth) - scaledWidth) / 2
	offsetY := (float64(windowHeight) - scaledHeight) / 2

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(offsetX, offsetY)

	screen.DrawImage(g.frame, op)
}

func (g *ebitenGame) Layout(outsideWidth int, outsideHeight int) (int, int) {
	// return outsideWidth, outsideHeight
	s := ebiten.Monitor().DeviceScaleFactor()
	return int(float64(outsideWidth) * s), int(float64(outsideHeight) * s)
}
