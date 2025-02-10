import { gameInfos } from "./types";

async function f(server: string, method: string, route: string, body?: any) {
  const req = await fetch(`${server}/${route}`, {
    method: method,
    headers: {
      "Content-Type": "application/json",
    },
    body: body && method !== "GET" ? JSON.stringify(body) : undefined,
  });
  if (!req.ok) {
    throw new Error(req.statusText);
  }
  return req;
}

export async function GetGameInfos(server: string) {
  const r = await f(server, "GET", "games");
  const data = await r.json();
  try {
    const response = gameInfos.parse(data);
    return response;
  } catch (e) {
    console.error(e);
    throw e;
  }
}
