// Shared config-editing state for the editor pages (Sources, Targets, Boards,
// Mappings, Settings). One draft object per page keeps the loading/saving/
// import/upsert logic in one place.
import { api, ApiError } from "./api";
import type { Config, Meta } from "./types";

export interface SaveMsg {
  text: string;
  ok: boolean;
}

export class ConfigDraft {
  meta = $state<Meta | null>(null);
  cfg = $state<Config | null>(null);
  msg = $state<SaveMsg | null>(null);

  async init(): Promise<void> {
    const [m, c] = await Promise.all([api.meta(), api.config()]);
    this.meta = m;
    this.cfg = structuredClone(c);
  }

  async reload(): Promise<void> {
    this.cfg = structuredClone(await api.config());
  }

  /** Save the whole config (all pages edit the same document). */
  async save(): Promise<boolean> {
    if (!this.cfg) return false;
    this.msg = null;
    if (this.cfg.updates && !this.cfg.updates.repo?.trim()) {
      this.cfg.updates = undefined; // empty repo = feature off
    }
    try {
      await api.saveConfig(this.cfg);
      this.msg = { text: "Saved — connectors reloaded.", ok: true };
      return true;
    } catch (e) {
      this.msg = { text: e instanceof ApiError ? e.errors.join("\n") : String(e), ok: false };
      return false;
    }
  }

  /** Merge an uploaded section file (bindings/sources/targets/profiles). */
  async importSection(file: File): Promise<void> {
    this.msg = null;
    try {
      await api.importSection(file);
      await this.reload();
      this.msg = { text: "Section imported (upserted by id).", ok: true };
    } catch (e) {
      this.msg = { text: e instanceof ApiError ? e.errors.join("\n") : String(e), ok: false };
    }
  }

  /** Full-config YAML upload. */
  async importConfig(file: File): Promise<void> {
    this.msg = null;
    try {
      await api.importConfig(file);
      await this.reload();
      this.msg = { text: "Config imported & applied.", ok: true };
    } catch (e) {
      this.msg = { text: e instanceof ApiError ? e.errors.join("\n") : String(e), ok: false };
    }
  }
}
