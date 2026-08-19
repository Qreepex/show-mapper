// Type layer: re-exports the backend-generated wire types (single source of
// truth — edit the Go structs and run `make types`, never these files!) and
// adds the few UI-only shapes that have no backend equivalent.
//
//   generated/config.ts  ← internal/config structs (Config, Binding, …)
//   generated/core.ts    ← internal/core wire types (Event, Action, SnapshotConn, …)
//   generated/server.ts  ← internal/server/wire.go (Envelope, Snapshot)
export * from "./generated/config";
export * from "./generated/core";
export * from "./generated/server";
export * from "./generated/updater";

import type { ProfileConfig } from "./generated/config";
import type {
  ActionPreset,
  ActionReport,
  ControlKind,
  Event as CoreEvent,
  SnapshotConn,
  TypeInfo,
} from "./generated/core";

// ---- readable aliases used across the UI ---------------------------------

export type ConnectorState = SnapshotConn;
export type SourceEvent = CoreEvent;
export type TargetAction = ActionReport;

// ---- string-union conveniences (backend uses plain strings) --------------

export type Trigger = "pressed" | "released" | "hold" | "value";
export type Mode = "momentary" | "toggle";
export type ActionType = "command" | "value" | "fader";
export type ValueType = "int" | "float";

// ---- /api/meta (backend assembles this from registries; not one Go struct) -

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
  presets: ActionPreset[];
  customProfiles: Record<string, ProfileConfig[]>;
}
