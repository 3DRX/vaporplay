import { GamepadAndState, GamepadState } from "@/lib/types";
import { isEqual } from "lodash";
import { useEffect, useRef } from "react";

export const defaultGamepadState: GamepadState = {
  connected: false,
  buttons: [],
  axes: [],
  id: "",
};

export default function useGamepad(props: {
  onGamepadStateChange?: (gamepadState: GamepadState, gamepad: Gamepad) => void;
}) {
  const gamepadRef = useRef<GamepadAndState>({
    gamepad: undefined,
    gamepadState: defaultGamepadState,
  });
  useEffect(() => {
    let animationFrameId: number;

    const handleGamepadConnected = (event: GamepadEvent) => {
      console.log("Gamepad connected:", event.gamepad);
    };

    const handleGamepadDisconnected = (event: GamepadEvent) => {
      console.log("Gamepad disconnected:", event.gamepad);
    };

    const updateGamepadState = () => {
      const gamepads = navigator.getGamepads();
      const activeGamepad = gamepads[0]; // Using first gamepad

      if (activeGamepad && gamepadRef.current) {
        const newGamepadState: GamepadState = {
          connected: true,
          buttons: Array.from(activeGamepad.buttons).map((button) => ({
            pressed: button.pressed,
            value: parseFloat(button.value.toFixed(2)),
          })),
          axes: Array.from(activeGamepad.axes).map((v) =>
            parseFloat(v.toFixed(2)),
          ),
          id: activeGamepad.id,
        };
        if (!isEqual(newGamepadState, gamepadRef.current.gamepadState)) {
          gamepadRef.current.gamepadState = newGamepadState;
          gamepadRef.current.gamepad = activeGamepad;
          props.onGamepadStateChange?.(
            gamepadRef.current.gamepadState,
            gamepadRef.current.gamepad,
          );
        }
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
  }, [props]);
}
