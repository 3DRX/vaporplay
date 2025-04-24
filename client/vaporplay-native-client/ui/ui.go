package ui

import (
	"bytes"
	"image/color"
	"log"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/image/font/gofont/goregular"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	clientconfig "github.com/3DRX/vaporplay/client/vaporplay-native-client/client-config"
)

type UIThread struct {
	startGamePromise chan *clientconfig.ClientConfig
	game             *ebitenGame
}

type ebitenGame struct {
	ui    *ebitenui.UI
	frame *ebiten.Image

	lock sync.Mutex
}

func NewUIThread() (*UIThread, chan *clientconfig.ClientConfig) {
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("VaporPlay")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetVsyncEnabled(false)

	// This creates the root container for this UI.
	// All other UI elements must be added to this container.
	rootContainer := widget.NewContainer()

	// This adds the root container to the UI, so that it will be rendered.
	eui := &ebitenui.UI{
		Container: rootContainer,
	}

	s, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		log.Fatal(err)
	}

	fontFace := &text.GoTextFace{
		Source: s,
		Size:   32,
	}
	// This creates a text widget that says "Hello World!"
	helloWorldLabel := widget.NewText(
		widget.TextOpts.Text("Hello World!", fontFace, color.White),
	)

	// To display the text widget, we have to add it to the root container.
	rootContainer.AddChild(helloWorldLabel)

	game := ebitenGame{
		ui: eui,
	}

	startGamePromise := make(chan *clientconfig.ClientConfig)

	return &UIThread{
		startGamePromise: startGamePromise,
		game:             &game,
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
	// ui.Update() must be called in ebiten Update function, to handle user input and other things
	// g.ui.Update()
	return nil
}

func (g *ebitenGame) Draw(screen *ebiten.Image) {
	// ui.Draw() should be called in the ebiten Draw function, to draw the UI onto the screen.
	// It should also be called after all other rendering for your game so that it shows up on top of your game world.
	// g.ui.Draw(screen)

	// draw the frame
	g.lock.Lock()
	defer g.lock.Unlock()

	var windowWidth, windowHeight int

	if ebiten.IsFullscreen() {
		windowWidth, windowHeight = ebiten.Monitor().Size()
	} else {
		windowWidth, windowHeight = ebiten.WindowSize()
	}
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
	return outsideWidth, outsideHeight
	// s := ebiten.Monitor().DeviceScaleFactor()
	// return int(float64(outsideWidth) * s), int(float64(outsideHeight) * s)
}
