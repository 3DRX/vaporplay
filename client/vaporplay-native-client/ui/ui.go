package ui

import (
	"image"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/ebitenui/ebitenui"
	"github.com/hajimehoshi/ebiten/v2"

	clientconfig "github.com/3DRX/vaporplay/client/vaporplay-native-client/client-config"
)

type UIThread struct {
	frameChan        <-chan image.Image
	startGamePromise chan *clientconfig.ClientConfig
	game             *ebitenGame
	configPath       *string
}

type ebitenGame struct {
	frame              *ebiten.Image
	closeWindowPromise chan<- struct{}
	ui                 *ebitenui.UI

	lock sync.Mutex
}

type ListEntry struct {
	id    int
	name  string
	value string
}

func NewUIThread(
	frameChan <-chan image.Image,
	configPath *string,
	closeWindowPromise chan<- struct{},
) (*UIThread, chan *clientconfig.ClientConfig) {
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("VaporPlay")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetVsyncEnabled(false)
	ebiten.SetWindowClosingHandled(true)

	game := &ebitenGame{
		closeWindowPromise: closeWindowPromise,
		ui:                 loadUI(*configPath),
	}

	startGamePromise := make(chan *clientconfig.ClientConfig)

	return &UIThread{
		frameChan:        frameChan,
		startGamePromise: startGamePromise,
		game:             game,
		configPath:       configPath,
	}, startGamePromise
}

func (u *UIThread) Spin() {
	go func() {
		for {
			img := <-u.frameChan
			u.game.lock.Lock()
			u.game.frame = ebiten.NewImageFromImage(img)
			u.game.lock.Unlock()
		}
	}()

	err := ebiten.RunGame(u.game)
	if err != nil {
		slog.Error("ebiten error", "error", err)
	}
}

func (g *ebitenGame) Update() error {
	g.lock.Lock()
	defer g.lock.Unlock()

	if ebiten.IsWindowBeingClosed() {
		slog.Info("closing window")
		g.closeWindowPromise <- struct{}{}
		time.Sleep(100 * time.Millisecond)
		os.Exit(0)
	}
	if g.frame == nil {
		g.ui.Update()
	}
	return nil
}

func (g *ebitenGame) Draw(screen *ebiten.Image) {
	g.lock.Lock()
	defer g.lock.Unlock()

	if g.frame == nil {
		// draw connection form
		g.ui.Draw(screen)
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
