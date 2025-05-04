package ui

import (
	"fmt"
	"image"
	"image/color"
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

type ListEntry struct {
	id   int
	name string
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
	serverLabel := widget.NewText(
		widget.TextOpts.Text("Server URL", face, hexToColor(textIdleColor)),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionStart,
			}),
		),
	)
	root.AddChild(serverLabel)
	serverInput := widget.NewTextInput(
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
	btnImg := &widget.ButtonImage{
		Idle:    eimage.NewNineSliceColor(color.NRGBA{R: 170, G: 170, B: 180, A: 255}),
		Hover:   eimage.NewNineSliceColor(color.NRGBA{R: 130, G: 130, B: 150, A: 255}),
		Pressed: eimage.NewNineSliceColor(color.NRGBA{R: 100, G: 100, B: 120, A: 255}),
	}
	ent := []ListEntry{
		{
			id:   0,
			name: "H.264 NVENC",
		},
		{
			id:   1,
			name: "H.265 NVENC",
		},
		{
			id:   2,
			name: "AV1 NVENC",
		},
		{
			id:   3,
			name: "x264",
		},
	}
	entries := make([]any, 0, len(ent))
	for _, e := range ent {
		entries = append(entries, e)
	}
	comboBox := widget.NewListComboButton(
		widget.ListComboButtonOpts.SelectComboButtonOpts(
			widget.SelectComboButtonOpts.ComboButtonOpts(
				//Set the max height of the dropdown list
				widget.ComboButtonOpts.MaxContentHeight(150),
				//Set the parameters for the primary displayed button
				widget.ComboButtonOpts.ButtonOpts(
					widget.ButtonOpts.Image(btnImg),
					widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(5)),
					widget.ButtonOpts.Text("", face, &widget.ButtonTextColor{
						Idle:     hexToColor(textIdleColor),
						Disabled: hexToColor(textDisabledColor),
					}),
					widget.ButtonOpts.WidgetOpts(
						//Set how wide the button should be
						widget.WidgetOpts.MinSize(150, 0),
						//Set the combobox's position
						widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
							HorizontalPosition: widget.AnchorLayoutPositionCenter,
							VerticalPosition:   widget.AnchorLayoutPositionCenter,
						})),
				),
			),
		),
		widget.ListComboButtonOpts.ListOpts(
			//Set how wide the dropdown list should be
			widget.ListOpts.ContainerOpts(widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(150, 0))),
			//Set the entries in the list
			widget.ListOpts.Entries(entries),
			widget.ListOpts.ScrollContainerOpts(
				//Set the background images/color for the dropdown list
				widget.ScrollContainerOpts.Image(&widget.ScrollContainerImage{
					Idle:     eimage.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
					Disabled: eimage.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
					Mask:     eimage.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
				}),
			),
			widget.ListOpts.SliderOpts(
				//Set the background images/color for the background of the slider track
				widget.SliderOpts.Images(&widget.SliderTrackImage{
					Idle:  eimage.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
					Hover: eimage.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
				}, btnImg),
				widget.SliderOpts.MinHandleSize(5),
				//Set how wide the track should be
				widget.SliderOpts.TrackPadding(widget.NewInsetsSimple(2))),
			//Set the font for the list options
			widget.ListOpts.EntryFontFace(face),
			//Set the colors for the list
			widget.ListOpts.EntryColor(&widget.ListEntryColor{
				Selected:                   color.NRGBA{254, 255, 255, 255},             //Foreground color for the unfocused selected entry
				Unselected:                 color.NRGBA{254, 255, 255, 255},             //Foreground color for the unfocused unselected entry
				SelectedBackground:         color.NRGBA{R: 130, G: 130, B: 200, A: 255}, //Background color for the unfocused selected entry
				SelectedFocusedBackground:  color.NRGBA{R: 130, G: 130, B: 170, A: 255}, //Background color for the focused selected entry
				FocusedBackground:          color.NRGBA{R: 170, G: 170, B: 180, A: 255}, //Background color for the focused unselected entry
				DisabledUnselected:         color.NRGBA{100, 100, 100, 255},             //Foreground color for the disabled unselected entry
				DisabledSelected:           color.NRGBA{100, 100, 100, 255},             //Foreground color for the disabled selected entry
				DisabledSelectedBackground: color.NRGBA{100, 100, 100, 255},             //Background color for the disabled selected entry
			}),
			//Padding for each entry
			widget.ListOpts.EntryTextPadding(widget.NewInsetsSimple(5)),
		),
		//Define how the entry is displayed
		widget.ListComboButtonOpts.EntryLabelFunc(
			func(e any) string {
				//Button Label function
				return "Button: " + e.(ListEntry).name
			},
			func(e any) string {
				//List Label function
				return "List: " + e.(ListEntry).name
			}),
		//Callback when a new entry is selected
		widget.ListComboButtonOpts.EntrySelectedHandler(func(args *widget.ListComboButtonEntrySelectedEventArgs) {
			fmt.Println("Selected Entry: ", args.Entry)
		}),
	)
	comboBox.SetSelectedEntry(entries[0])
	root.AddChild(comboBox)

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
