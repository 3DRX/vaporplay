import useGamepad from "@/hooks/use-gamepad";
import { Button } from "@/components/ui/button";

export default function GamepadTest() {
  const { gamepadState: gamepad, gamepad: gamepadObj } = useGamepad();

  return (
    <div className="p-4">
      <h2 className="mb-4 text-xl font-bold">Gamepad Status</h2>
      <div className="space-y-4">
        <div className="flex">
          <p>
            Connection Status:{" "}
            {gamepad.connected ? "Connected" : "Disconnected"}
          </p>
          <div className="grow" />
          {gamepad.connected && gamepad.id && (
            <p className="text-muted-foreground">{gamepad.id}</p>
          )}
          {!gamepad.connected && (
            <p className="text-muted-foreground">
              You need to touch a button on your controller for this to pick it
              up.
            </p>
          )}
        </div>

        {gamepad.connected && (
          <>
            <div>
              <h3 className="mb-2 font-semibold">Buttons:</h3>
              <div className="grid grid-cols-4 gap-2 text-black">
                {gamepad.buttons.map((button, index) => (
                  <div
                    key={index}
                    className={`rounded p-2 ${
                      button.pressed ? "bg-blue-500 text-white" : "bg-gray-200"
                    }`}
                  >
                    Button {index}: {button.value.toFixed(2)}
                  </div>
                ))}
              </div>
            </div>

            <div>
              <h3 className="mb-2 font-semibold">Axes:</h3>
              <div className="grid grid-cols-2 gap-2 text-black">
                {gamepad.axes.map((axis, index) => (
                  <div key={index} className="rounded bg-gray-200 p-2">
                    Axis {index}: {axis.toFixed(2)}
                  </div>
                ))}
              </div>
            </div>

            <div>
              <h3 className="mb-2 font-semibold">Vibrator:</h3>
              <div>
                <Button
                  variant="outline"
                  onMouseDown={() => {
                    if (gamepadObj && gamepadObj.vibrationActuator) {
                      gamepadObj.vibrationActuator.playEffect("dual-rumble", {
                        duration: 100,
                        strongMagnitude: 1.0,
                        weakMagnitude: 1.0,
                      });
                    }
                  }}
                >
                  Vibrate
                </Button>
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
