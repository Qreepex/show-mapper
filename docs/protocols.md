# API & realtime protocol

Base URL: same as the UI (default `http://127.0.0.1:8080`). Everything is JSON.
The web UI in `web/` uses exactly these endpoints — treat this file as the
contract; change code + types (`web/src/lib/types.ts`) + this doc together.

## REST

| Method | Path | Body → Response | Purpose |
|---|---|---|---|
| GET | `/api/health` | → `{ok, version, clients}` | liveness + ws client count |
| GET | `/api/meta` | → `{version, commit, sourceTypes[], targetTypes[], triggers, modes, actionTypes, controlKinds, ledStyles, customProfiles}` | everything the UI needs to render schema-agnostically |
| GET | `/api/config` | → `Config` | current config |
| PUT | `/api/config` | `Config` → `{ok:true}` or `400 {ok:false, errors:[...]}` | validate → persist (atomic) → hot-reload connectors → broadcast `config.updated`. Unknown JSON fields are rejected. |
| GET | `/api/state` | → `{connectors: [{id, kind, type, status{state, detail}, controls?}], configPath}` | runtime connector snapshot |
| GET | `/api/sources/{type}/inspect` | → `{ok:true, result:{...}}` or `{ok:false, error}` | connector-provided enumeration. For `midi`: `result = {in:[{number,name}], out:[…]}`; non-CGO builds return ok:false with a hint. 404 if the type has no inspector. |

`GET /` (anything non-`/api`, non-`/ws`) serves the embedded SPA
(adapter-static output; history fallback to `200.html`).

## WebSocket `/ws`

- Wire format (all messages): `{ "type": "<type>", "ts": "<RFC3339>", "data": {...} }`.
- Origin policy: Origin header (if present) must match Host. The UI is served
  same-origin and vite proxies `/ws` in dev, so nothing to configure.
- On connect the server immediately sends **`state.snapshot`**:
  `{version, commit, connectors[], config}`.
- Slow consumers are dropped (bounded per-client queue).

### Server → client message types

| type | data | when |
|---|---|---|
| `state.snapshot` | `{version, commit, connectors[], config}` | once after connect |
| `source.event` | `{source, control, kind: pressed\|released\|value, value: 0..1, raw, when}` | every physical control change (incl. unmapped `note:N`/`cc:N`) |
| `target.action` | `{binding, ok, error?, action{target, kind, address, args}}` | after each dispatched action (ok=false on send failure) |
| `connector.status` | `{id, kind: source\|target, type, status{state, detail}}` | connect/error/connected transitions |
| `config.updated` | `Config` | after a successful `PUT /api/config` (any client) |

### Client → server

Reserved namespace `client.*` (e.g. future `client.test-binding`,
`client.led-test`). Currently ignored; the read loop exists to keep
pings/pongs alive.

## Field/enum values

Single source of truth is the backend (`/api/meta` mirrors them):
`triggers: [pressed, released, hold, value]` · `modes: [momentary, toggle]` ·
`actionTypes: [command, value, fader]` · `valueType: int|float` ·
`controlKinds: [pad, button, fader, encoder]` ·
`ledStyles: [none, onOff, velocity, apc2-rgb]` · status states:
`connecting|connected|disconnected|error`.

## Worked example

```bash
curl http://127.0.0.1:8080/api/config > cfg.json
jq '.bindings += [{source:"wing", control:"pad-0-0", trigger:"pressed",
  target:"ma3", action:{type:"command", address:"/cmd", command:"Go Executor 1.201"}}]' \
  cfg.json > cfg2.json
curl -X PUT -H 'Content-Type: application/json' --data @cfg2.json \
  http://127.0.0.1:8080/api/config   # → {"ok":true} — live-reloaded
# watch:
wscat -c ws://127.0.0.1:8080/ws
```
