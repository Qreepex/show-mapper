// Shared types mirroring the Go backend JSON shapes (internal/config + internal/core).
// Keep in sync with docs/protocols.md.

export type Trigger = "pressed" | "released" | "hold" | "value";
export type Mode = "momentary" | "toggle";
export type ActionType = "command" | "value" | "fader";
export type ValueType = "int" | "float";
export type ControlKind = "pad" | "button" | "fader" | "encoder";

export interface HTTPConfig {
  listen: string;
}

export interface SourceConfig {
  id: string;
  type: string;
  profile?: string;
  options?: Record<string, unknown>;
}

export interface TargetConfig {
  id: string;
  type: string;
  options?: Record<string, unknown>;
}

export interface LEDConfig {
  color?: string;
  mode?: string;
}

export interface ActionConfig {
  type: ActionType;
  address: string;
  command?: string;
  releaseCommand?: string;
  pressValue?: number | null;
  releaseValue?: number | null;
  range?: [number, number] | null;
  valueType?: ValueType;
}

export interface Binding {
  id?: string;
  source: string;
  control: string;
  trigger: Trigger;
  holdMs?: number;
  mode?: Mode;
  target: string;
  action: ActionConfig;
  led?: LEDConfig | null;
}

export interface ProfileLED {
  style?: string;
  onVelocity?: number;
  colors?: Record<string, number>;
}

export interface ProfileControl {
  id: string;
  label: string;
  kind: ControlKind;
  row?: number | null;
  col?: number | null;
  note?: number | null;
  cc?: number | null;
  hasLED?: boolean;
  ledNote?: number | null;
}

export interface ProfileConfig {
  id: string;
  type: string;
  name: string;
  match?: string[];
  led?: ProfileLED;
  controls: ProfileControl[];
}

export interface Config {
  version: number;
  http: HTTPConfig;
  profiles?: ProfileConfig[];
  sources: SourceConfig[];
  targets: TargetConfig[];
  bindings: Binding[];
}

// ---- /api/meta ----

export interface Control {
  id: string;
  label: string;
  kind: ControlKind;
  row?: number;
  col?: number;
  hasLED: boolean;
}

export interface ProfileSummary {
  id: string;
  name: string;
  led: string;
  controls: Control[];
}

export interface FieldSpec {
  name: string;
  label: string;
  type: "text" | "number";
  required?: boolean;
  default?: unknown;
  help?: string;
}

export interface TypeInfo {
  type: string;
  name: string;
  options: FieldSpec[];
  profiles?: ProfileSummary[];
}

export interface Meta {
  version: string;
  commit: string;
  sourceTypes: TypeInfo[];
  targetTypes: TypeInfo[];
  triggers: Trigger[];
  modes: Mode[];
  actionTypes: ActionType[];
  controlKinds: ControlKind[];
  ledStyles: string[];
  customProfiles: Record<string, ProfileConfig[]>;
}

// ---- realtime state ----

export interface ConnStatus {
  state: "connected" | "connecting" | "disconnected" | "error";
  detail: string;
}

export interface ConnectorState {
  id: string;
  kind: "source" | "target";
  type: string;
  status: ConnStatus;
  controls?: Control[];
}

export interface Envelope<T = unknown> {
  type: string;
  ts: string;
  data: T;
}

export interface SnapshotData {
  version: string;
  commit: string;
  connectors: ConnectorState[];
  config: Config;
}

export interface SourceEvent {
  source: string;
  control: string;
  kind: "pressed" | "released" | "value";
  value: number;
  raw: number;
  when: string;
}

export interface TargetAction {
  binding: string;
  ok: boolean;
  error?: string;
  action: {
    binding: string;
    target: string;
    kind: ActionType;
    address: string;
    args: unknown[];
  };
}
