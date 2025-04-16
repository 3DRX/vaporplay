import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import { GamepadState, GamepadStateDto } from "@/lib/types";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function toGamepadStateDto(gamepad: GamepadState): GamepadStateDto {
  return {
    b: gamepad.buttons.map((button) => button.value),
    a: Array.from(gamepad.axes),
  };
}
