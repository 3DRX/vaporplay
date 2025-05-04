package ui

import (
	"fmt"
	"image/color"
	"strconv"

	"github.com/ebitenui/ebitenui"
	eimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

func loadUI() *ebitenui.UI {
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
		widget.TextInputOpts.SubmitHandler(func(args *widget.TextInputChangedEventArgs) {
			fmt.Println("Server URL: ", args.InputText)
		}),
	)
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
		}),
	)
	codecComboBox.SetSelectedEntry(entries[0])
	codecCfgContainer.AddChild(codecComboBox)
	ent = []ListEntry{
		{
			id:   0,
			name: "30 FPS",
		},
		{
			id:   1,
			name: "60 FPS",
		},
		{
			id:   2,
			name: "90 FPS",
		},
		{
			id:   3,
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
		}),
	)
	fpsComboBox.SetSelectedEntry(entries[0])
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
			args.TextInput.Submit()
		}),
		widget.TextInputOpts.SubmitHandler(func(args *widget.TextInputChangedEventArgs) {
			fmt.Println("Initial Rate: ", args.InputText)
		}),
	)
	initRateInput.SetText("5")
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
			args.TextInput.Submit()
		}),
		widget.TextInputOpts.SubmitHandler(func(args *widget.TextInputChangedEventArgs) {
			fmt.Println("Initial Rate: ", args.InputText)
		}),
	)
	maxRateInput.SetText("30")
	codecCfgContainer.AddChild(maxRateInput)
	root.AddChild(codecCfgContainer)
	ui := &ebitenui.UI{
		Container: root,
	}
	return ui
}
