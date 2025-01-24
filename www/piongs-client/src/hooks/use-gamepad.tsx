import { useEffect, useState } from "react";

export interface GamepadState {
  connected: boolean;
  buttons: {
    pressed: boolean;
    value: number;
  }[];
  axes: number[];
  id: string;
}

export default function useGamepad() {
  const [gamepad, setGamepad] = useState<Gamepad | undefined>(undefined);
  const [gamepadState, setGamepadState] = useState<GamepadState>({
    connected: false,
    buttons: [],
    axes: [],
    id: "",
  });

  useEffect(() => {
    let animationFrameId: number;

    const handleGamepadConnected = (event: GamepadEvent) => {
      console.log("Gamepad connected:", event.gamepad);
    };

    const handleGamepadDisconnected = (event: GamepadEvent) => {
      console.log("Gamepad disconnected:", event.gamepad);
      setGamepadState((prev) => ({ ...prev, connected: false }));
    };

    const updateGamepadState = () => {
      const gamepads = navigator.getGamepads();
      const activeGamepad = gamepads[0]; // Using first gamepad

      if (activeGamepad) {
        setGamepadState({
          connected: true,
          buttons: Array.from(activeGamepad.buttons).map((button) => ({
            pressed: button.pressed,
            value: button.value,
          })),
          axes: Array.from(activeGamepad.axes),
          id: activeGamepad.id,
        });
        setGamepad(activeGamepad);
      }

      animationFrameId = requestAnimationFrame(updateGamepadState);
    };

    window.addEventListener("gamepadconnected", handleGamepadConnected);
    window.addEventListener("gamepaddisconnected", handleGamepadDisconnected);

    // Start polling for gamepad state
    animationFrameId = requestAnimationFrame(updateGamepadState);

    return () => {
      window.removeEventListener("gamepadconnected", handleGamepadConnected);
      window.removeEventListener(
        "gamepaddisconnected",
        handleGamepadDisconnected,
      );
      cancelAnimationFrame(animationFrameId);
    };
  }, []);

  return { gamepad, gamepadState };
}
