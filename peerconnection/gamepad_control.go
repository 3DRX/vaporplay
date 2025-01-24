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
