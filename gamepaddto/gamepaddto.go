package gamepaddto

type GamepadDTO struct {
	Buttons []float32 `json:"b"`
	Axes    []float32 `json:"a"`
}
