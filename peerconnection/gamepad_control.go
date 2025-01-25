package peerconnection

import "github.com/bendahl/uinput"

type GamepadControl struct {
	Gamepad uinput.Gamepad
}

type GamepadDTO struct {
	Buttons []float32 `json:"b"`
	Axes    []float32 `json:"a"`
}

func NewGamepadControl() (*GamepadControl, error) {
	gamepad, err := uinput.CreateGamepad("/dev/uinput", []byte("Xbox Wireless Controller"), 0x045e, 0x0b13)
	if err != nil {
		return nil, err
	}
	return &GamepadControl{
		Gamepad: gamepad,
	}, nil
}

var ButtonMap = map[int]int{
	0: uinput.ButtonSouth,
	1: uinput.ButtonEast,
	2: uinput.ButtonWest,
	3: uinput.ButtonNorth,

	4: uinput.ButtonBumperLeft,
	5: uinput.ButtonBumperRight,

	8: uinput.ButtonSelect,
	9: uinput.ButtonStart,

	10: uinput.ButtonThumbLeft,
	11: uinput.ButtonThumbRight,

	12: uinput.ButtonDpadUp,
	13: uinput.ButtonDpadDown,
	14: uinput.ButtonDpadLeft,
	15: uinput.ButtonDpadRight,

	16: uinput.ButtonMode,
	17: uinput.ButtonGamepad,
}

var HatMap = map[int]uinput.HatDirection{
	// TODO: github.com/bendahl/uinput don't support trigger axis,
	// so we nned to fork it and implement it some time in the future.
	6: uinput.HatLeft,
	7: uinput.HatRight,
}

func (g *GamepadControl) SendGamepadState(dto *GamepadDTO) {
	for i, v := range dto.Buttons {
		if v > 0 {
			uinputButton, ok := ButtonMap[i]
			if ok {
				g.Gamepad.ButtonDown(uinputButton)
			}
			uinputHat, ok := HatMap[i]
			if ok {
				g.Gamepad.HatPress(uinputHat)
			}
		} else {
			uinputButton, ok := ButtonMap[i]
			if ok {
				g.Gamepad.ButtonUp(uinputButton)
			}
			uinputHat, ok := HatMap[i]
			if ok {
				g.Gamepad.HatRelease(uinputHat)
			}
		}
	}
	g.Gamepad.LeftStickMove(dto.Axes[0], dto.Axes[1])
	g.Gamepad.RightStickMove(dto.Axes[2], dto.Axes[3])
}

func (g *GamepadControl) Close() {
	g.Gamepad.Close()
}
