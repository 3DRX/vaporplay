package ui

import (
	"image"
	"log/slog"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	clientconfig "github.com/3DRX/vaporplay/client/vaporplay-native-client/client-config"
	"github.com/3DRX/vaporplay/config"
)

type UIThread struct {
	frameChan        <-chan image.Image
	startGamePromise chan *clientconfig.ClientConfig
	game             *ebitenGame
}

type ebitenGame struct {
	frame *ebiten.Image

	lock sync.Mutex
}

func NewUIThread(frameChan <-chan image.Image) (*UIThread, chan *clientconfig.ClientConfig) {
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("VaporPlay")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetVsyncEnabled(false)

	game := &ebitenGame{}

	startGamePromise := make(chan *clientconfig.ClientConfig)

	return &UIThread{
		frameChan:        frameChan,
		startGamePromise: startGamePromise,
		game:             game,
	}, startGamePromise
}

func (u *UIThread) Spin() {
	// imgGenerator := VideoRecord()
	go func() {
		// ticker := time.NewTicker(8300 * time.Microsecond)
		for {
			img := <-u.frameChan
			// img := imgGenerator()
			u.game.lock.Lock()
			u.game.frame = ebiten.NewImageFromImage(img)
			u.game.lock.Unlock()
		}
	}()

	go u.readConfig()

	err := ebiten.RunGame(u.game)
	if err != nil {
		slog.Error("ebiten error", "error", err)
	}
}

func (u *UIThread) readConfig() {
	// wait for 2 seconds, mock user input
	time.Sleep(2 * time.Second)

	cfg := &clientconfig.ClientConfig{
		Addr: "192.168.2.213:8080",
		SessionConfig: config.SessionConfig{
			GameConfig: config.GameConfig{
				GameId:          "383870",
				GameWindowName:  "Firewatch",
				GameDisplayName: "Fire Watch",
				GameIcon:        "",
				EndGameCommands: []config.KillProcessCommandConfig{{
					Flags:       []string{},
					ProcessName: "fw.x86_64",
				}},
			},
			CodecConfig: config.CodecConfig{
				Codec:          "av1_nvenc",
				InitialBitrate: 10_000_000,
				MaxBitrate:     20_000_000,
				FrameRate:      60,
			},
		},
	}
	slog.Info("start game", "game_id", cfg.SessionConfig.GameConfig.GameId)
	u.startGamePromise <- cfg
}

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
