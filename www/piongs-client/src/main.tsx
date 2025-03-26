import { StrictMode, useCallback, useState } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import { ThemeProvider } from "@/components/theme-provider";
import ConnectionForm from "@/components/connection-form";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { CodecInfoType, FormType, GameInfoType } from "@/lib/types";
import Gameplay from "@/components/gameplay";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import GamepadTest from "@/components/gamepad-test";
import { BrowserRouter, Link, Route, Routes } from "react-router";
import { Button } from "./components/ui/button";
import CodecCapabilities from "./components/codec-capabilities";
import { useLocalStorage } from "@uidotdev/usehooks";

const queryClient = new QueryClient();

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme-piongs">
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <Routes>
            <Route path="/" element={<App />} />
            <Route path="/gamepad-test" element={<GamepadTest />} />
            <Route path="/codec-capabilities" element={<CodecCapabilities />} />
          </Routes>
        </BrowserRouter>
      </QueryClientProvider>
    </ThemeProvider>
  </StrictMode>,
);

function App() {
  const [startGame, setStartGame] = useState(false);
  const [server, setServer] = useLocalStorage("piongs-client-server", "");
  const [codec, setCodec] = useLocalStorage<CodecInfoType>(
    "piongs-client-codec",
    {
      codec: "h264_nvenc",
      initial_bitrate: 5_000_000,
      frame_rate: 60,
      max_bitrate: 30_000_000,
    },
  );
  const [game, setGame] = useState<GameInfoType | null>(null);
  const [record, setRecord] = useLocalStorage("piongs-client-record", false);

  function onSubmit(values: FormType) {
    if (values.server) {
      setServer(values.server);
    }
    if (values.game) {
      setGame(values.game);
    }
    setCodec({
      codec: values.codec,
      frame_rate: values.frame_rate,
      initial_bitrate: values.initial_bitrate,
      max_bitrate: values.max_bitrate,
    });
    setStartGame(true);
  }

  function onFirstSubmit(values: FormType) {
    if (values.server) {
      setServer(values.server);
    }
    setCodec({
      codec: values.codec,
      frame_rate: values.frame_rate,
      initial_bitrate: values.initial_bitrate,
      max_bitrate: values.max_bitrate,
    });
    setRecord(values.record);
  }

  const onExit = useCallback(() => setStartGame(false), []);

  return (
    <div className="max-h-svh">
      {server.length !== 0 && startGame && game ? (
        <Gameplay
          server={server}
          game={game}
          codec={codec}
          onExit={onExit}
          record={record}
        />
      ) : (
        <>
          <Card className="mx-auto mt-10 max-w-[60rem]">
            <CardHeader>
              <CardTitle>Connect to server</CardTitle>
              <CardDescription></CardDescription>
            </CardHeader>
            <CardContent>
              <ConnectionForm
                defaultServer={server}
                defaultCodec={codec}
                defaultRecord={record}
                onSubmit={onSubmit}
                onFirstSubmit={onFirstSubmit}
              />
            </CardContent>
          </Card>
          <div className="mx-auto mt-10 max-w-[60rem]">
            <h2 className="text-xl font-bold">Debugging</h2>
            <ul className="my-5">
              <li>
                <Button variant="link" className="underline">
                  <Link to="/gamepad-test">Gamepad Test</Link>
                </Button>
              </li>
              <li>
                <Button variant="link" className="underline">
                  <Link to="/codec-capabilities">Codec Capabilities</Link>
                </Button>
              </li>
            </ul>
          </div>
        </>
      )}
    </div>
  );
}
