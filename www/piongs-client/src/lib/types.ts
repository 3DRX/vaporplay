import { z } from "zod";

export const gameInfo = z.object({
  game_id: z.string().nonempty(),
  game_window_name: z.string().nonempty(),
  game_display_name: z.string().nonempty(),
  game_icon: z.string(), // the base64 encoded icon, optional
  game_process_name: z.string().nonempty(),
});

export const gameInfos = z.array(gameInfo);

export type GameInfoType = z.infer<typeof gameInfo>;

export const formSchema = z.object({
  server: z.string().nonempty(),
  game: gameInfo,
});

export type FormType = z.infer<typeof formSchema>;

export interface GamepadState {
  connected: boolean;
  buttons: {
    pressed: boolean;
    value: number;
  }[];
  axes: number[];
  id: string;
}

export interface GamepadAndState {
  gamepad: Gamepad | undefined;
  gamepadState: GamepadState;
}

export type GamepadStateDto = {
  b: number[]; // buttons, > 0 is pressed, 0 is unpressed
  a: number[]; // axes
};
