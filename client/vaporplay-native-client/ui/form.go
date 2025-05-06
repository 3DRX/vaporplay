package ui

import (
	"fmt"
	"image/color"
	"log/slog"
	"strconv"
	"time"

	clientconfig "github.com/3DRX/vaporplay/client/vaporplay-native-client/client-config"
	"github.com/3DRX/vaporplay/config"
	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/image"
	eimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

type ListEntry struct {
	id    int
	name  string
	value string
}

type GameListEntry struct {
	id string
}

func loadUI(configPath string, startGamePromise chan *clientconfig.ClientConfig) *ebitenui.UI {
	cfg := useClientConfig(configPath)
	face, err := loadFont(22)
	if err != nil {
		panic(err)
	}
	smallFace, err := loadFont(20)
	if err != nil {
		panic(err)
	}
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
		widget.TextInputOpts.ChangedHandler(func(args *widget.TextInputChangedEventArgs) {
			setClientConfig(configPath, func(cc *clientconfig.ClientConfig) {
				cc.Addr = args.InputText
			})
		}),
	)
	serverInput.SetText(cfg.Addr)
	root.AddChild(serverInput)
	codecCfgContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
				widget.RowLayoutOpts.Spacing(20),
				widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(20)),
			),
		),
	)
	btnImg := &widget.ButtonImage{
		Idle:    eimage.NewNineSliceColor(color.NRGBA{R: 170, G: 170, B: 180, A: 255}),
		Hover:   eimage.NewNineSliceColor(color.NRGBA{R: 130, G: 130, B: 150, A: 255}),
		Pressed: eimage.NewNineSliceColor(color.NRGBA{R: 100, G: 100, B: 120, A: 255}),
	}
	ent := []ListEntry{
		{
			id:    0,
			name:  "H.264 NVENC",
			value: "h264_nvenc",
		},
		{
			id:    1,
			name:  "H.265 NVENC",
			value: "hevc_nvenc",
		},
		{
			id:    2,
			name:  "AV1 NVENC",
			value: "av1_nvenc",
		},
		{
			id:    3,
			name:  "x264",
			value: "libx264",
		},
	}
	entries := make([]any, 0, len(ent))
	for _, e := range ent {
		entries = append(entries, e)
	}
	codecComboBox := widget.NewListComboButton(
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
				return e.(ListEntry).name
			},
			func(e any) string {
				//List Label function
				return e.(ListEntry).name
			}),
		//Callback when a new entry is selected
		widget.ListComboButtonOpts.EntrySelectedHandler(func(args *widget.ListComboButtonEntrySelectedEventArgs) {
			fmt.Println("Selected Codec: ", args.Entry)
			listEntry, ok := args.Entry.(ListEntry)
			if !ok {
				return
			}
			setClientConfig(configPath, func(cc *clientconfig.ClientConfig) {
				cc.SessionConfig.CodecConfig.Codec = listEntry.value
			})
		}),
	)
	switch cfg.SessionConfig.CodecConfig.Codec {
	case "h264_nvenc":
		codecComboBox.SetSelectedEntry(entries[0])
	case "hevc_nvenc":
		codecComboBox.SetSelectedEntry(entries[1])
	case "av1_nvenc":
		codecComboBox.SetSelectedEntry(entries[2])
	case "libx264":
		codecComboBox.SetSelectedEntry(entries[3])
	default:
		slog.Warn("unknown codec: " + cfg.SessionConfig.CodecConfig.Codec)
		codecComboBox.SetSelectedEntry(entries[0])
	}
	codecCfgContainer.AddChild(codecComboBox)
	ent = []ListEntry{
		{
			id:   30,
			name: "30 FPS",
		},
		{
			id:   60,
			name: "60 FPS",
		},
		{
			id:   90,
			name: "90 FPS",
		},
		{
			id:   120,
			name: "120 FPS",
		},
	}
	entries = make([]any, 0, len(ent))
	for _, e := range ent {
		entries = append(entries, e)
	}
	fpsComboBox := widget.NewListComboButton(
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
				return e.(ListEntry).name
			},
			func(e any) string {
				//List Label function
				return e.(ListEntry).name
			}),
		//Callback when a new entry is selected
		widget.ListComboButtonOpts.EntrySelectedHandler(func(args *widget.ListComboButtonEntrySelectedEventArgs) {
			fmt.Println("Selected FPS: ", args.Entry)
			listEntry, ok := args.Entry.(ListEntry)
			if !ok {
				return
			}
			setClientConfig(configPath, func(cc *clientconfig.ClientConfig) {
				cc.SessionConfig.CodecConfig.FrameRate = float32(listEntry.id)
			})
		}),
	)
	se := entries[1]
	for i, v := range ent {
		if v.id == int(cfg.SessionConfig.CodecConfig.FrameRate) {
			se = entries[i]
			break
		}
	}
	fpsComboBox.SetSelectedEntry(se)
	codecCfgContainer.AddChild(fpsComboBox)
	initRateLabel := widget.NewText(
		widget.TextOpts.Text("Initial Rate (Mbps)", smallFace, hexToColor(labelIdleColor)),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}),
		),
	)
	codecCfgContainer.AddChild(initRateLabel)
	lastInitRateInputText := ""
	initRateInput := widget.NewTextInput(
		widget.TextInputOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
				Stretch:  true,
			}),
			widget.WidgetOpts.MinSize(150, 0),
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
		widget.TextInputOpts.ChangedHandler(func(args *widget.TextInputChangedEventArgs) {
			// try to parse input text as number or empty string,
			// if not, restore to last state
			if args.InputText == "" {
				lastInitRateInputText = ""
				return
			}
			a, err := strconv.Atoi(args.InputText)
			if err != nil || a < 0 {
				args.TextInput.SetText(lastInitRateInputText)
				return
			}
			lastInitRateInputText = args.InputText
			setClientConfig(configPath, func(cc *clientconfig.ClientConfig) {
				cc.SessionConfig.CodecConfig.InitialBitrate = a * 1_000_000
			})
		}),
	)
	initRateInput.SetText(fmt.Sprintf("%d", cfg.SessionConfig.CodecConfig.InitialBitrate/1_000_000))
	codecCfgContainer.AddChild(initRateInput)
	maxRateLabel := widget.NewText(
		widget.TextOpts.Text("Max Rate (Mbps)", smallFace, hexToColor(labelIdleColor)),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}),
		),
	)
	codecCfgContainer.AddChild(maxRateLabel)
	lastMaxRateInputText := ""
	maxRateInput := widget.NewTextInput(
		widget.TextInputOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
				Stretch:  true,
			}),
			widget.WidgetOpts.MinSize(150, 0),
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
		widget.TextInputOpts.ChangedHandler(func(args *widget.TextInputChangedEventArgs) {
			// try to parse input text as number or empty string,
			// if not, restore to last state
			if args.InputText == "" {
				lastInitRateInputText = ""
				return
			}
			a, err := strconv.Atoi(args.InputText)
			if err != nil || a < 0 {
				args.TextInput.SetText(lastMaxRateInputText)
				return
			}
			lastMaxRateInputText = args.InputText
			setClientConfig(configPath, func(cc *clientconfig.ClientConfig) {
				cc.SessionConfig.CodecConfig.MaxBitrate = a * 1_000_000
			})
		}),
	)
	maxRateInput.SetText(fmt.Sprintf("%d", cfg.SessionConfig.CodecConfig.MaxBitrate/1_000_000))
	codecCfgContainer.AddChild(maxRateInput)
	root.AddChild(codecCfgContainer)
	nextButton := widget.NewButton(
		// set general widget options
		widget.ButtonOpts.WidgetOpts(
			// instruct the container's anchor layout to center the button both horizontally and vertically.
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionEnd,
				Stretch:  false,
			}),
		),
		// specify the images to use.
		widget.ButtonOpts.Image(loadButtonImage()),
		// specify the button's text, the font face, and the color.
		widget.ButtonOpts.Text("Next", face, &widget.ButtonTextColor{
			Idle:     hexToColor(textIdleColor),
			Disabled: hexToColor(textDisabledColor),
		}),
		// specify that the button's text needs some padding for correct display.
		widget.ButtonOpts.TextPadding(widget.Insets{
			Left:   30,
			Right:  30,
			Top:    5,
			Bottom: 5,
		}),
		// Move the text down and right on press
		widget.ButtonOpts.PressedHandler(func(args *widget.ButtonPressedEventArgs) {
			args.Button.Text().Padding.Top = 1
			args.Button.Text().Padding.Bottom = -1
			args.Button.GetWidget().CustomData = true
		}),
		// Move the text back to start on press released
		widget.ButtonOpts.ReleasedHandler(func(args *widget.ButtonReleasedEventArgs) {
			args.Button.Text().Padding.Top = 0
			args.Button.Text().Padding.Bottom = 0
			args.Button.GetWidget().CustomData = false
		}),
		// add a handler that reacts to clicking the button.
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			args.Button.GetWidget().Visibility = widget.Visibility_Hide

			// fetch game data from server
			time.Sleep(1 * time.Second)
			gameconfigs, err := fetchGameData(configPath)
			if err != nil {
				slog.Error("failed to fetch game data from server", "error", err)
				// go back to previous state
				args.Button.GetWidget().Visibility = widget.Visibility_Show
				return
			}
			gameconfigsMap := make(map[string]config.GameConfig)
			entries = make([]any, 0, len(ent))
			for i := range *gameconfigs {
				e := (*gameconfigs)[i]
				gameconfigsMap[e.GameId] = e
				entries = append(entries, GameListEntry{
					id: e.GameId,
				})
			}
			gameComboBox := widget.NewListComboButton(
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
								widget.WidgetOpts.MinSize(500, 0),
								widget.WidgetOpts.LayoutData(widget.RowLayoutData{
									Position: widget.RowLayoutPositionCenter,
									Stretch:  false,
								})),
						),
					),
				),
				widget.ListComboButtonOpts.ListOpts(
					//Set how wide the dropdown list should be
					widget.ListOpts.ContainerOpts(
						widget.ContainerOpts.WidgetOpts(
							widget.WidgetOpts.MinSize(500, 0),
						),
					),
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
						return gameconfigsMap[e.(GameListEntry).id].GameDisplayName
					},
					func(e any) string {
						//List Label function
						return gameconfigsMap[e.(GameListEntry).id].GameDisplayName
					}),
				//Callback when a new entry is selected
				widget.ListComboButtonOpts.EntrySelectedHandler(func(args *widget.ListComboButtonEntrySelectedEventArgs) {
					gameConfig, ok := gameconfigsMap[args.Entry.(GameListEntry).id]
					if !ok {
						return
					}
					setClientConfig(configPath, func(cc *clientconfig.ClientConfig) {
						cc.SessionConfig.GameConfig = gameConfig
					})
				}),
			)
			_, ok := gameconfigsMap[cfg.SessionConfig.GameConfig.GameId]
			if ok {
				gameComboBox.SetSelectedEntry(GameListEntry{
					id: cfg.SessionConfig.GameConfig.GameId,
				})
			} else {
				setClientConfig(configPath, func(cc *clientconfig.ClientConfig) {
					cc.SessionConfig.GameConfig = (*gameconfigs)[0]
				})
			}
			root.AddChild(gameComboBox)
			submitButton := widget.NewButton(
				// set general widget options
				widget.ButtonOpts.WidgetOpts(
					// instruct the container's anchor layout to center the button both horizontally and vertically.
					widget.WidgetOpts.LayoutData(widget.RowLayoutData{
						Position: widget.RowLayoutPositionEnd,
						Stretch:  false,
					}),
				),
				// specify the images to use.
				widget.ButtonOpts.Image(loadButtonImage()),
				// specify the button's text, the font face, and the color.
				widget.ButtonOpts.Text("Start !", face, &widget.ButtonTextColor{
					Idle:     hexToColor(textIdleColor),
					Disabled: hexToColor(textDisabledColor),
				}),
				// specify that the button's text needs some padding for correct display.
				widget.ButtonOpts.TextPadding(widget.Insets{
					Left:   30,
					Right:  30,
					Top:    5,
					Bottom: 5,
				}),
				// Move the text down and right on press
				widget.ButtonOpts.PressedHandler(func(args *widget.ButtonPressedEventArgs) {
					args.Button.Text().Padding.Top = 1
					args.Button.Text().Padding.Bottom = -1
					args.Button.GetWidget().CustomData = true
				}),
				// Move the text back to start on press released
				widget.ButtonOpts.ReleasedHandler(func(args *widget.ButtonReleasedEventArgs) {
					args.Button.Text().Padding.Top = 0
					args.Button.Text().Padding.Bottom = 0
					args.Button.GetWidget().CustomData = false
				}),
				// add a handler that reacts to clicking the button.
				widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
					clientConfig := useClientConfig(configPath)
					slog.Info("submit", "client config", clientConfig)
					startGamePromise <- clientConfig
				}),
				// Indicate that this button should not be submitted when enter or space are pressed
				widget.ButtonOpts.DisableDefaultKeys(),
			)
			root.AddChild(submitButton)
		}),
		// Indicate that this button should not be submitted when enter or space are pressed
		widget.ButtonOpts.DisableDefaultKeys(),
	)
	root.AddChild(nextButton)
	ui := &ebitenui.UI{
		Container: root,
	}
	return ui
}

func loadButtonImage() *widget.ButtonImage {
	idle := eimage.NewBorderedNineSliceColor(color.NRGBA{R: 170, G: 170, B: 180, A: 255}, color.NRGBA{90, 90, 90, 255}, 3)
	hover := eimage.NewBorderedNineSliceColor(color.NRGBA{R: 130, G: 130, B: 150, A: 255}, color.NRGBA{70, 70, 70, 255}, 3)
	pressed := eimage.NewAdvancedNineSliceColor(color.NRGBA{R: 130, G: 130, B: 150, A: 255}, image.NewBorder(3, 2, 2, 2, color.NRGBA{70, 70, 70, 255}))

	return &widget.ButtonImage{
		Idle:    idle,
		Hover:   hover,
		Pressed: pressed,
	}
}
