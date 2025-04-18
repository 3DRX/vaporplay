package uinput

import (
	"errors"
	"fmt"
	"io"
	"os"
)

const MaximumAxisValue = 32767

// HatDirection specifies the direction of hat movement
type HatDirection int

const (
	HatUp HatDirection = iota + 1
	HatDown
	HatLeft
	HatRight
)

type HatAction int

const (
	Press HatAction = iota + 1
	Release
)

// Gamepad is a hybrid key / absolute change event output device.
// It used to enable a program to simulate gamepad input events.
type Gamepad interface {
	// ButtonPress will cause the button to be pressed and immediately released.
	ButtonPress(key int) error

	// ButtonDown will send a button-press event to an existing gamepad device.
	// The key can be any of the predefined keycodes from keycodes.go.
	// Note that the key will be "held down" until "KeyUp" is called.
	ButtonDown(key int) error

	// ButtonUp will send a button-release event to an existing gamepad device.
	// The key can be any of the predefined keycodes from keycodes.go.
	ButtonUp(key int) error

	// LeftStickMoveX performs a movement of the left stick along the x-axis
	LeftStickMoveX(value float32) error
	// LeftStickMoveY performs a movement of the left stick along the y-axis
	LeftStickMoveY(value float32) error

	// RightStickMoveX performs a movement of the right stick along the x-axis
	RightStickMoveX(value float32) error
	// RightStickMoveY performs a movement of the right stick along the y-axis
	RightStickMoveY(value float32) error

	// LeftStickMove moves the left stick along the x and y-axis
	LeftStickMove(x, y float32) error
	// RightStickMove moves the right stick along the x and y-axis
	RightStickMove(x, y float32) error

	// HatPress will issue a hat-press event in the given direction
	HatPress(direction HatDirection) error
	// HatRelease will issue a hat-release event in the given direction
	HatRelease(direction HatDirection) error

	// LeftTriggerForce performs a trigger-axis-z event with a given force
	LeftTriggerForce(value float32) error
	// RightTriggerForce performs a trigger-axis-rz event with a given force
	RightTriggerForce(value float32) error

	io.Closer
}

type GamepadWithRumble interface {
	Gamepad
	// Call this function periodically to check for force-feedback.
	// the callback return will be placed into upload.ReturnValue
	// it is not an guarante that the callback will be called
	ForceFeedbackCallback(callback func(upload *UInputFFUpload, erase *UInputFFErase) int32) error
}

type vGamepad struct {
	name       []byte
	deviceFile *os.File
}

// CreateGamepad will create a new gamepad using the given uinput
// device path of the uinput device.
func CreateGamepad(path string, name []byte, vendor uint16, product uint16) (Gamepad, error) { // TODO: Consider moving this to a generic function that works for all devices
	err := validateDevicePath(path)
	if err != nil {
		return nil, err
	}
	err = validateUinputName(name)
	if err != nil {
		return nil, err
	}

	fd, err := createVGamepadDevice(path, name, vendor, product, 0)
	if err != nil {
		return nil, err
	}

	return vGamepad{name: name, deviceFile: fd}, nil
}

// CreateGamepadWithRumble will create a new gamepad using the given uinput
// device path of the uinput device, and will rumble support.
// Using a gamepad with rumble requires calling ForceFeedbackCallback periodically
func CreateGamepadWithRumble(path string, name []byte, vendor uint16, product uint16, effectsMax uint32) (GamepadWithRumble, error) {
	err := validateDevicePath(path)
	if err != nil {
		return nil, err
	}
	err = validateUinputName(name)
	if err != nil {
		return nil, err
	}

	if effectsMax < 1 {
		return nil, fmt.Errorf("effectsMax is below the minimum value of 1, use CreateGamepad if you don't want rumble support")
	}

	fd, err := createVGamepadDevice(path, name, vendor, product, effectsMax)
	if err != nil {
		return nil, err
	}

	return vGamepad{name: name, deviceFile: fd}, nil
}

func (vg vGamepad) ButtonPress(key int) error {
	err := vg.ButtonDown(key)
	if err != nil {
		return err
	}
	err = vg.ButtonUp(key)
	if err != nil {
		return err
	}
	return nil
}

func (vg vGamepad) ButtonDown(key int) error {
	return sendBtnEvent(vg.deviceFile, []int{key}, btnStatePressed)
}

func (vg vGamepad) ButtonUp(key int) error {
	return sendBtnEvent(vg.deviceFile, []int{key}, btnStateReleased)
}

func (vg vGamepad) LeftStickMoveX(value float32) error {
	return vg.sendStickAxisEvent(absX, value)
}

func (vg vGamepad) LeftStickMoveY(value float32) error {
	return vg.sendStickAxisEvent(absY, value)
}

func (vg vGamepad) RightStickMoveX(value float32) error {
	return vg.sendStickAxisEvent(absRX, value)
}

func (vg vGamepad) RightStickMoveY(value float32) error {
	return vg.sendStickAxisEvent(absRY, value)
}

func (vg vGamepad) LeftStickMove(x, y float32) error {
	values := map[uint16]float32{}
	values[absX] = x
	values[absY] = y

	return vg.sendStickEvent(values)
}

func (vg vGamepad) RightStickMove(x, y float32) error {
	values := map[uint16]float32{}
	values[absRX] = x
	values[absRY] = y

	return vg.sendStickEvent(values)
}

func (vg vGamepad) LeftTriggerForce(value float32) error {
	return vg.sendStickAxisEvent(absZ, value)
}

func (vg vGamepad) RightTriggerForce(value float32) error {
	return vg.sendStickAxisEvent(absRZ, value)
}

func (vg vGamepad) HatPress(direction HatDirection) error {
	return vg.sendHatEvent(direction, Press)
}

func (vg vGamepad) HatRelease(direction HatDirection) error {
	return vg.sendHatEvent(direction, Release)
}

func (vg vGamepad) ForceFeedbackCallback(callback func(upload *UInputFFUpload, erase *UInputFFErase) int32) error {
	return forceFeedbackCallback(vg.deviceFile, callback)
}

func (vg vGamepad) sendStickAxisEvent(absCode uint16, value float32) error {
	ev := inputEvent{
		Type:  evAbs,
		Code:  absCode,
		Value: denormalizeInput(value),
	}

	buf, err := inputEventToBuffer(ev)
	if err != nil {
		return fmt.Errorf("writing abs stick event failed: %v", err)
	}

	_, err = vg.deviceFile.Write(buf)
	if err != nil {
		return fmt.Errorf("failed to write abs stick event to device file: %v", err)
	}

	return syncEvents(vg.deviceFile)
}

func (vg vGamepad) sendStickEvent(values map[uint16]float32) error {
	for code, value := range values {
		ev := inputEvent{
			Type:  evAbs,
			Code:  code,
			Value: denormalizeInput(value),
		}

		buf, err := inputEventToBuffer(ev)
		if err != nil {
			return fmt.Errorf("writing abs stick event failed: %v", err)
		}

		_, err = vg.deviceFile.Write(buf)
		if err != nil {
			return fmt.Errorf("failed to write abs stick event to device file: %v", err)
		}
	}

	return syncEvents(vg.deviceFile)
}

func (vg vGamepad) sendHatEvent(direction HatDirection, action HatAction) error {
	var event uint16
	var value int32

	switch direction {
	case HatUp:
		{
			event = absHat0Y
			value = -1
		}
	case HatDown:
		{
			event = absHat0Y
			value = 1
		}
	case HatLeft:
		{
			event = absHat0X
			value = -1
		}
	case HatRight:
		{
			event = absHat0X
			value = 1
		}
	default:
		{
			return errors.New("failed to parse input direction")
		}
	}

	if action == Release {
		value = 0
	}

	ev := inputEvent{
		Type:  evAbs,
		Code:  event,
		Value: value,
	}

	buf, err := inputEventToBuffer(ev)
	if err != nil {
		return fmt.Errorf("writing abs stick event failed: %v", err)
	}

	_, err = vg.deviceFile.Write(buf)
	if err != nil {
		return fmt.Errorf("failed to write abs stick event to device file: %v", err)
	}

	return syncEvents(vg.deviceFile)
}

func (vg vGamepad) Close() error {
	return closeDevice(vg.deviceFile)
}

func createVGamepadDevice(path string, name []byte, vendor uint16, product uint16, effMax uint32) (fd *os.File, err error) {
	// This array is needed to register the event keys for the gamepad device.
	keys := []uint16{
		ButtonGamepad,

		ButtonSouth,
		ButtonEast,
		ButtonNorth,
		ButtonWest,

		ButtonBumperLeft,
		ButtonBumperRight,
		ButtonTriggerLeft,
		ButtonTriggerRight,
		ButtonThumbLeft,
		ButtonThumbRight,

		ButtonSelect,
		ButtonStart,

		ButtonDpadUp,    // * * *
		ButtonDpadDown,  // * These buttons can be used instead of the hat events.
		ButtonDpadLeft,  // *
		ButtonDpadRight, // * * *

		ButtonMode,
	}

	// absEvents is for the absolute events for the gamepad device.
	absEvents := []uint16{
		absX,
		absY,
		absZ,
		absRX,
		absRY,
		absRZ,
		absHat0X,
		absHat0Y,
	}

	ffEvents := []uint16{
		FFRumble,
	}

	// tell uinput what the minimum/maximum abs value is
	var absMin [absSize]int32
	absMin[absX] = -MaximumAxisValue
	absMin[absY] = -MaximumAxisValue
	absMin[absZ] = -MaximumAxisValue
	absMin[absRX] = -MaximumAxisValue
	absMin[absRY] = -MaximumAxisValue
	absMin[absRZ] = -MaximumAxisValue
	absMin[absHat0X] = -MaximumAxisValue
	absMin[absHat0Y] = -MaximumAxisValue

	var absMax [absSize]int32
	absMax[absX] = MaximumAxisValue
	absMax[absY] = MaximumAxisValue
	absMax[absZ] = MaximumAxisValue
	absMax[absRX] = MaximumAxisValue
	absMax[absRY] = MaximumAxisValue
	absMax[absRZ] = MaximumAxisValue
	absMax[absHat0X] = MaximumAxisValue
	absMax[absHat0Y] = MaximumAxisValue

	deviceFile, err := createDeviceFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create virtual gamepad device: %v", err)
	}

	// register button events
	err = registerDevice(deviceFile, uintptr(evKey))
	if err != nil {
		_ = deviceFile.Close()
		return nil, fmt.Errorf("failed to register virtual gamepad device: %v", err)
	}

	for _, code := range keys {
		err = ioctl(deviceFile, uiSetKeyBit, uintptr(code))
		if err != nil {
			_ = deviceFile.Close()
			return nil, fmt.Errorf("failed to register key number %d: %v", code, err)
		}
	}

	// register absolute events
	err = registerDevice(deviceFile, uintptr(evAbs))
	if err != nil {
		_ = deviceFile.Close()
		return nil, fmt.Errorf("failed to register absolute event input device: %v", err)
	}

	for _, event := range absEvents {
		err = ioctl(deviceFile, uiSetAbsBit, uintptr(event))
		if err != nil {
			_ = deviceFile.Close()
			return nil, fmt.Errorf("failed to register absolute event %v: %v", event, err)
		}
	}

	// register force-feedback events
	if effMax > 0 {
		err = registerDevice(deviceFile, uintptr(evFF))
		if err != nil {
			_ = deviceFile.Close()
			return nil, fmt.Errorf("failed to register ff event input device: %v", err)
		}
		for _, event := range ffEvents {
			err = ioctl(deviceFile, uiSetFFBit, uintptr(event))
			if err != nil {
				_ = deviceFile.Close()
				return nil, fmt.Errorf("failed to register ff event %v: %v", FFRumble, err)
			}
		}
	}

	return createUsbDevice(deviceFile,
		uinputUserDev{
			Name: toUinputName(name),
			ID: inputID{
				Bustype: busUsb,
				Vendor:  vendor,
				Product: product,
				Version: 1,
			},
			EffectsMax: effMax,
			Absmin:     absMin,
			Absmax:     absMax,
		})
}

// Takes in a normalized value (-1.0:1.0) and return an event value
func denormalizeInput(value float32) int32 {
	return int32(value * MaximumAxisValue)
}
