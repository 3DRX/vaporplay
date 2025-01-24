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
import { FormType, GameInfoType } from "@/lib/types";
import Gameplay from "@/components/gameplay";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import GamepadTest from "@/components/gamepad-test";
import { BrowserRouter, Link, Route, Routes } from "react-router";
import { Button } from "./components/ui/button";

const queryClient = new QueryClient();

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <Routes>
            <Route path="/" element={<App />} />
            <Route path="/gamepad-test" element={<GamepadTest />} />
          </Routes>
        </BrowserRouter>
      </QueryClientProvider>
    </ThemeProvider>
  </StrictMode>,
);

function App() {
  const [server, setServer] = useState("");
  const [game, setGame] = useState<GameInfoType | null>(null);

  function onSubmit(values: FormType) {
    if (values.server) {
      setServer(values.server);
    }
    if (values.game) {
      setGame(values.game);
    }
  }

  const onExit = useCallback(() => setServer(""), []);

  return (
    <div className="min-h-screen">
      {server.length !== 0 && game ? (
        <Gameplay server={server} game={game} onExit={onExit} />
      ) : (
        <>
          <Card className="mx-auto mt-10 max-w-[60rem]">
            <CardHeader>
              <CardTitle>Connect to server</CardTitle>
              <CardDescription></CardDescription>
            </CardHeader>
            <CardContent>
              <ConnectionForm onSubmit={onSubmit} />
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
            </ul>
          </div>
        </>
      )}
    </div>
  );
}
