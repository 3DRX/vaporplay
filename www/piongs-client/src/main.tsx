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

const queryClient = new QueryClient();

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
      <QueryClientProvider client={queryClient}>
        <App />
      </QueryClientProvider>
    </ThemeProvider>
  </StrictMode>,
);

function App() {
  const [server, setServer] = useState("");
  const [game, setGame] = useState<GameInfoType | null>(null);

  function onSubmit(values: FormType) {
    console.log("============");
    console.log(values);
    console.log("============");
    if (values.server) {
      setServer(values.server);
    }
    if (values.game) {
      setGame(values.game);
    }
  }

  const onExit = useCallback(() => setServer(""), []);

  return (
    <>
      {server.length !== 0 && game ? (
        <Gameplay server={server} game={game} onExit={onExit} />
      ) : (
        <Card className="mx-auto mt-10 max-w-[60rem]">
          <CardHeader>
            <CardTitle>Connect to server</CardTitle>
            <CardDescription></CardDescription>
          </CardHeader>
          <CardContent>
            <ConnectionForm onSubmit={onSubmit} />
          </CardContent>
        </Card>
      )}
    </>
  );
}
