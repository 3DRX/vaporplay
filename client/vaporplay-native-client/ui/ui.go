package ui

import (
	"fmt"
	"image"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/ebitenui/ebitenui"
	eimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
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

	face, _ := loadFont(22)
	root := widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Spacing(20),
				widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(20)),
			),
		),
	)
	serverInput := widget.NewTextInput(
		widget.TextInputOpts.Placeholder("Server URL"),
		widget.TextInputOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
				Stretch:  true,
			}),
		),
		widget.TextInputOpts.Image(&widget.TextInputImage{
			Idle:     eimage.NewNineSliceColor(hexToColor(backgroundColor)),
			Disabled: eimage.NewNineSliceColor(hexToColor(backgroundColor)),
		}),
		widget.TextInputOpts.Face(face),
		widget.TextInputOpts.Color(&widget.TextInputColor{
			Idle:          hexToColor(textIdleColor),
			Disabled:      hexToColor(textDisabledColor),
			Caret:         hexToColor(textInputCaretColor),
			DisabledCaret: hexToColor(textInputDisabledCaretColor),
		}),
		widget.TextInputOpts.Padding(widget.NewInsetsSimple(5)),
		widget.TextInputOpts.CaretOpts(
			widget.CaretOpts.Size(face, 2),
		),
		widget.TextInputOpts.SubmitHandler(func(args *widget.TextInputChangedEventArgs) {
			fmt.Println("Server URL: ", args.InputText)
		}),
	)
	root.AddChild(serverInput)

	game := &ebitenGame{
		closeWindowPromise: closeWindowPromise,
		ui: &ebitenui.UI{
			Container: root,
		},
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

	go u.readConfig()

	err := ebiten.RunGame(u.game)
	if err != nil {
		slog.Error("ebiten error", "error", err)
	}
}

func (u *UIThread) readConfig() {
	// wait 1 second, mock user input
	// time.Sleep(1 * time.Second)
	// cfg := clientconfig.LoadClientConfig(u.configPath)
	// slog.Info("start game", "game_id", cfg.SessionConfig.GameConfig.GameId)
	// u.startGamePromise <- cfg
}

func (g *ebitenGame) Update() error {
	g.lock.Lock()
	defer g.lock.Unlock()

	if ebiten.IsWindowBeingClosed() {
		slog.Info("closing window")
		g.closeWindowPromise <- struct{}{}
		time.Sleep(1 * time.Second)
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
		screen.Fill(image.Black)
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
