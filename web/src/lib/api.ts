// Thin REST client for the showbridge API (see docs/protocols.md).
import type { Config, ConnectorState, Meta } from "./types";

export class ApiError extends Error {
  errors: string[];
  constructor(errors: string[]) {
    super(errors.join("; "));
    this.errors = errors;
  }
}

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, init);
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    const errs = Array.isArray(body.errors) ? (body.errors as string[]) : [res.statusText];
    throw new ApiError(errs);
  }
  return body as T;
}

export const api = {
  meta: () => req<Meta>("/api/meta"),
  config: () => req<Config>("/api/config"),
  state: () => req<{ connectors: ConnectorState[]; configPath: string }>("/api/state"),
  saveConfig: (cfg: Config) =>
    req<{ ok: boolean }>("/api/config", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(cfg),
    }),
  inspectSource: (type: string) =>
    req<{ ok: boolean; error?: string; result?: { in?: { number: number; name: string }[]; out?: { number: number; name: string }[] } }>(
      `/api/sources/${type}/inspect`,
    ),
};
