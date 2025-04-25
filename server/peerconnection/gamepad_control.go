package peerconnection

import (
	"github.com/3DRX/vaporplay/gamepaddto"
	"github.com/3DRX/vaporplay/uinput"
)

type GamepadControl struct {
	Gamepad uinput.Gamepad
}

func NewGamepadControl() (*GamepadControl, error) {
	gamepad, err := uinput.CreateGamepad("/dev/uinput", []byte("Vaporplay Virtual Controller"), 0x9999, 0x999)
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
	2: uinput.ButtonNorth,
	3: uinput.ButtonWest,

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
}

func (g *GamepadControl) SendGamepadState(dto *gamepaddto.GamepadDTO) {
	for i, v := range dto.Buttons {
		if v > 0 {
			uinputButton, ok := ButtonMap[i]
			if ok {
				g.Gamepad.ButtonDown(uinputButton)
			}
		} else {
			uinputButton, ok := ButtonMap[i]
			if ok {
				g.Gamepad.ButtonUp(uinputButton)
			}
		}
	}
	g.Gamepad.LeftStickMove(dto.Axes[0], dto.Axes[1])
	g.Gamepad.RightStickMove(dto.Axes[2], dto.Axes[3])
	g.Gamepad.LeftTriggerForce(dto.Buttons[6]*2 - 1)
	g.Gamepad.RightTriggerForce(dto.Buttons[7]*2 - 1)
}

func (g *GamepadControl) Close() error {
	return g.Gamepad.Close()
}
