// Realtime state built on Svelte 5 runes — a class holding $state mirrors of
// the backend's WebSocket stream. No stores (project rule: runes only).
import type {
  Config,
  ConnectorState,
  Envelope,
  SnapshotData,
  SourceEvent,
  TargetAction,
} from "./types";

export type WsState = "connecting" | "open" | "reconnecting" | "closed";

export interface TickerEntry {
  ts: Date;
  kind: string;
  text: string;
  ok?: boolean;
}

const TICKER_MAX = 200;

function summarize(msg: Envelope): TickerEntry | null {
  const ts = new Date(msg.ts);
  switch (msg.type) {
    case "source.event": {
      const d = msg.data as SourceEvent;
      const val =
        d.kind === "value" ? ` ${(d.value * 100).toFixed(0)}% (raw ${d.raw})` : "";
      return { ts, kind: "source.event", text: `${d.source} · ${d.control} → ${d.kind}${val}` };
    }
    case "target.action": {
      const d = msg.data as TargetAction;
      const args = d.action.args.map((a) => JSON.stringify(a)).join(", ");
      const text = d.ok
        ? `${d.binding} → ${d.action.target} ${d.action.address} ${args}`
        : `${d.binding} ✗ ${d.error ?? "send failed"}`;
      return { ts, kind: "target.action", text, ok: d.ok };
    }
    case "connector.status": {
      const d = msg.data as { id: string; kind: string; status: { state: string; detail: string } };
      return { ts, kind: "connector.status", text: `${d.kind} ${d.id}: ${d.status.state} — ${d.status.detail}` };
    }
    default:
      return null;
  }
}

class Live {
  ws = $state<WsState>("closed");
  version = $state("");
  commit = $state("");
  config = $state<Config | null>(null);
  connectors = $state<ConnectorState[]>([]);
  ticker = $state<TickerEntry[]>([]);

  #socket: WebSocket | null = null;
  #retries = 0;

  connect() {
    if (this.#socket) return;
    this.ws = this.#retries > 0 ? "reconnecting" : "connecting";
    const scheme = location.protocol === "https:" ? "wss" : "ws";
    const sock = new WebSocket(`${scheme}://${location.host}/ws`);
    this.#socket = sock;

    sock.onopen = () => {
      this.ws = "open";
      this.#retries = 0;
    };
    sock.onclose = () => {
      this.#socket = null;
      this.ws = "closed";
      this.#scheduleReconnect();
    };
    sock.onerror = () => {
      // onclose follows and handles reconnect
    };
    sock.onmessage = (ev) => this.#onMessage(ev);
  }

  #scheduleReconnect() {
    this.#retries++;
    const delay = Math.min(5000, 500 * this.#retries);
    setTimeout(() => this.connect(), delay);
  }

  #onMessage(ev: MessageEvent) {
    let msg: Envelope;
    try {
      msg = JSON.parse(ev.data as string) as Envelope;
    } catch {
      return;
    }
    switch (msg.type) {
      case "state.snapshot": {
        const d = msg.data as SnapshotData;
        this.version = d.version;
        this.commit = d.commit;
        this.config = d.config;
        this.connectors = d.connectors;
        break;
      }
      case "connector.status": {
        const d = msg.data as ConnectorState;
        const i = this.connectors.findIndex((c) => c.id === d.id && c.kind === d.kind);
        if (i >= 0) {
          this.connectors[i] = { ...this.connectors[i], status: d.status };
        } else {
          this.connectors = [...this.connectors, d];
        }
        break;
      }
      case "config.updated": {
        this.config = msg.data as Config;
        break;
      }
    }
    const entry = summarize(msg);
    if (entry) {
      this.ticker = [entry, ...this.ticker].slice(0, TICKER_MAX);
    }
  }

  connector(id: string): ConnectorState | undefined {
    return this.connectors.find((c) => c.id === id);
  }
}

export const live = new Live();
