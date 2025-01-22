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
import { FormType } from "@/lib/types";
import Gameplay from "@/components/gameplay";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
      <App />
    </ThemeProvider>
  </StrictMode>,
);

function App() {
  const [server, setServer] = useState("");
  function onSubmit(values: FormType) {
    console.log(values);
    setServer(values.server);
  }

  const onExit = useCallback(() => setServer(""), []);

  return (
    <>
      {server.length !== 0 ? (
        <Gameplay server={server} onExit={onExit} />
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
