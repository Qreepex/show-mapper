// Thin REST client for the show-mapper API (see docs/protocols.md).
import type { ActionConfig, Config, ConnectorState, Meta, NICInfo, UpdateStatus } from "./types";

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
  /** URL of the full-config YAML download (open in a tab / use as anchor href). */
  exportConfigURL: "/api/config/export",
  /** Upload a full YAML config (same schema as the download above). */
  importConfig: async (file: File) => {
    const res = await fetch("/api/config/import", {
      method: "POST",
      headers: { "Content-Type": "application/yaml" },
      body: await file.text(),
    });
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new ApiError(Array.isArray(body.errors) ? body.errors : [res.statusText]);
    }
  },
  /** Resolve a helper-module action preset (e.g. gma3.go) into an ActionConfig. */
  resolvePreset: (id: string, params: Record<string, unknown>) =>
    req<ActionConfig>("/api/presets/resolve", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ id, params }),
    }),
  exportSectionURL: (kind: "bindings" | "sources" | "targets" | "profiles") =>
    `/api/config/export/sections/${kind}`,
  /** Merge an uploaded section file into the config (upsert/replace). */
  importSection: async (file: File, mode: "upsert" | "replace" = "upsert") => {
    const res = await fetch(`/api/config/import/section?mode=${mode}`, {
      method: "POST",
      headers: { "Content-Type": "application/yaml" },
      body: await file.text(),
    });
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new ApiError(Array.isArray(body.errors) ? body.errors : [res.statusText]);
    }
  },
  /** Local network interfaces (for per-instance NIC bind options). */
  interfaces: () => req<{ interfaces: NICInfo[] }>("/api/system/interfaces"),
  /** Self-update (active when updates.repo is set in the config). */
  updateStatus: () => req<UpdateStatus>("/api/update/status"),
  updateCheck: () => req<UpdateStatus>("/api/update/check", { method: "POST" }),
  updateApply: () =>
    req<{ ok: boolean; message: string }>("/api/update/apply", { method: "POST" }),
  inspectSource: (type: string) =>
    req<{
      ok: boolean;
      error?: string;
      result?: { in?: { number: number; name: string }[]; out?: { number: number; name: string }[] };
    }>(`/api/sources/${type}/inspect`),
};
