import type { AddResult, ReposResponse } from "./types";

// The page talks to the trama server that served it and to nothing else.
async function call(path: string, init?: RequestInit): Promise<Response> {
  const res = await fetch(path, init);
  if (!res.ok) {
    let message = res.statusText;
    try {
      const body = await res.json();
      if (body && typeof body.error === "string") message = body.error;
    } catch {
      // not JSON: keep the status text
    }
    throw new Error(message);
  }
  return res;
}

export async function getRepos(org: string): Promise<ReposResponse> {
  const res = await call(org ? `/api/repos?org=${encodeURIComponent(org)}` : "/api/repos", { cache: "no-store" });
  return res.json();
}

function write(token: string, body: unknown): RequestInit {
  return {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-Trama-Token": token },
    body: JSON.stringify(body),
  };
}

export async function addRepos(token: string, org: string, ids: string[]): Promise<AddResult> {
  const res = await call("/api/add", write(token, { org, ids }));
  return res.json();
}

export async function cancel(token: string): Promise<void> {
  await call("/api/cancel", write(token, {}));
}
