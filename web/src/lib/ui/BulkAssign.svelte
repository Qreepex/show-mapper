<script lang="ts">
  // BulkAssign generates a range of bindings in one shot: pick a control
  // range (start + count over the source's ordered control list), one target
  // action, and let any numeric action parameter count up per binding
  // (e.g. 15 keyboard keys → executors 101–115 on page 4). Existing
  // bindings with the same source/control/trigger are skipped.
  import type { ActionPreset, Binding, Mode, Trigger } from "$lib/types";
  import { LED_COLORS, LED_MODES } from "$lib/options";
  import Button from "./Button.svelte";
  import Card from "./Card.svelte";
  import CheckboxInput from "./CheckboxInput.svelte";
  import Field from "./Field.svelte";
  import NumberInput from "./NumberInput.svelte";
  import SelectInput from "./SelectInput.svelte";
  import TextInput from "./TextInput.svelte";

  export interface ControlOpt {
    id: string;
    label: string;
    kind: string;
  }

  let {
    sources = [],
    targets = [],
    presets = [],
    triggers = [],
    modes = [],
    existing = [],
    controlsOf,
    targetTypeOf,
    onadd,
    onclose,
  }: {
    sources: { id: string }[];
    targets: { id: string }[];
    presets: ActionPreset[];
    triggers: Trigger[];
    modes: Mode[];
    existing: Binding[];
    controlsOf: (sourceId: string) => ControlOpt[];
    targetTypeOf: (targetId: string) => string;
    onadd: (bs: Binding[]) => void;
    onclose?: () => void;
  } = $props();

  // ---- control range ---------------------------------------------------------

  // Snapshot the first entries as defaults (props are stable by the time the
  // page renders this panel) — read once via closures.
  const initialSourceId = () => sources[0]?.id ?? "";
  const initialTargetId = () => targets[0]?.id ?? "";

  let source = $state(initialSourceId());
  let kind = $state(""); // "" = all kinds (e.g. MPK "Klaviatur" keys share kind "button")
  let startId = $state("");
  let count = $state(8);

  const controls = $derived(controlsOf(source));
  const kinds = $derived([...new Set(controls.map((c) => c.kind))]);
  const filtered = $derived(kind ? controls.filter((c) => c.kind === kind) : controls);
  // Empty/unknown startId → from the beginning of the list.
  const startIdx = $derived(Math.max(0, filtered.findIndex((c) => c.id === startId)));
  const selection = $derived(filtered.slice(startIdx, startIdx + Math.max(1, count)));

  // ---- target action (preset or raw template) --------------------------------

  const presetPool = (t: string) =>
    presets.filter((p) => !p.targetTypes || p.targetTypes.length === 0 || p.targetTypes.includes(targetTypeOf(t)));
  const p0 = presetPool(initialTargetId())[0];

  let target = $state(initialTargetId());
  let presetId = $state(p0?.id ?? "");
  const usablePresets = $derived(presetPool(target));
  // User pick, falling back to the first usable preset of the current target
  // (handles target switches silently pointing at an incompatible preset).
  const preset = $derived(usablePresets.find((p) => p.id === presetId) ?? usablePresets[0]);

  let params = $state<Record<string, string>>(initialParams(p0));
  let inc = $state<Record<string, boolean>>({});
  let step = $state<Record<string, string>>({});

  function initialParams(p?: ActionPreset): Record<string, string> {
    const out: Record<string, string> = {};
    for (const f of p?.fields ?? []) out[f.name] = String(f.default ?? "");
    return out;
  }

  function pickPreset(id: string) {
    presetId = id;
    const p = usablePresets.find((x) => x.id === id);
    params = initialParams(p);
    inc = {};
    step = {};
  }

  function setParam(name: string, v: string) {
    params = { ...params, [name]: v };
  }

  // Raw-action template when the target has no presets: {i} → 0-based index,
  // {n} → 1-based (e.g. "/Page4/Key{n}" with start control index in mind).
  let addressTpl = $state("/cmd");
  let commandTpl = $state("");

  function tpl(s: string, i: number): string {
    return s.replaceAll("{i}", String(i)).replaceAll("{n}", String(i + 1));
  }

  // ---- behavior (same for all generated bindings) ----------------------------

  let trigger = $state<Trigger>("pressed");
  let mode = $state<Mode>("momentary");
  let holdMs = $state(500);
  let ledColor = $state("green");
  let ledMode = $state("on");

  // ---- generation ------------------------------------------------------------

  function paramValue(field: string, base: string, i: number): string {
    if (!inc[field]) return base;
    const b = parseInt(base, 10);
    if (!Number.isFinite(b)) return base; // not numeric → constant
    const st = parseInt(step[field] ?? "1", 10);
    return String(b + i * (Number.isFinite(st) ? st : 1));
  }

  function genBindings(): Binding[] {
    return selection.map((c, i) => {
      const b: Binding = {
        source,
        control: c.id,
        trigger,
        mode,
        target,
        action: { type: "command", address: "/cmd", valueType: "int" },
        led: { color: ledColor, mode: ledMode },
      };
      if (trigger === "hold") b.holdMs = holdMs;
      if (preset) {
        const prm: Record<string, string> = {};
        for (const f of preset.fields) {
          prm[f.name] = paramValue(f.name, params[f.name] ?? String(f.default ?? ""), i);
        }
        b.action = { type: "preset", preset: preset.id, params: prm, address: "", valueType: "int" };
      } else {
        b.action = { type: "command", address: tpl(addressTpl, i) || "/cmd", command: tpl(commandTpl, i), valueType: "int" };
      }
      return b;
    });
  }

  const generated = $derived(genBindings());

  function keyOf(b: Binding): string {
    return `${b.source}/${b.control}/${b.trigger}`;
  }

  const preview = $derived(
    generated.map((b) => {
      const what =
        b.action.type === "preset"
          ? `${b.action.preset}  ${Object.entries(b.action.params ?? {})
              .map(([k, v]) => `${k}=${v}`)
              .join(" ")}`
          : `${b.action.address}${b.action.command ? `  ${b.action.command}` : ""}`;
      return `${b.control}  →  ${what}`;
    }),
  );
  const PREVIEW_MAX = 14;

  // ---- submit ----------------------------------------------------------------

  let resultMsg = $state("");

  function submit() {
    const seen = new Set(existing.map(keyOf));
    const out: Binding[] = [];
    let skipped = 0;
    for (const b of generated) {
      const k = keyOf(b);
      if (seen.has(k)) {
        skipped++;
        continue;
      }
      seen.add(k);
      out.push(b);
    }
    onadd(out);
    resultMsg =
      `Added ${out.length} binding${out.length === 1 ? "" : "s"}` +
      (skipped ? ` — skipped ${skipped} already mapped control${skipped === 1 ? "" : "s"}` : "") +
      ".";
  }
</script>

<Card title="Bulk assign controls">
  {#snippet actions()}
    {#if onclose}<Button variant="ghost" onclick={onclose}>✕ close</Button>{/if}
  {/snippet}

  <div class="rowline">
    <Field label="Source">
      <SelectInput value={source} options={sources.map((s) => s.id)}
        onchange={(e: Event) => { source = (e.currentTarget as HTMLSelectElement).value; }} />
    </Field>
    <Field label="Kind">
      <SelectInput value={kind} options={kinds} allowEmpty="(all)" 
        onchange={(e: Event) => { kind = (e.currentTarget as HTMLSelectElement).value; }} />
    </Field>
    <Field label="Start control" grow>
      <SelectInput value={startId} allowEmpty="(first control)"
        options={filtered.map((c) => ({ value: c.id, label: `${c.id} — ${c.label}` }))}
        onchange={(e: Event) => { startId = (e.currentTarget as HTMLSelectElement).value; }} />
    </Field>
    <Field label="Count" hint={`max ${filtered.length - startIdx} from here`}>
      <NumberInput min={1} max={Math.max(1, filtered.length - startIdx)} value={count}
        oninput={(e: Event) => { count = Number((e.currentTarget as HTMLInputElement).value); }} />
    </Field>
  </div>

  <div class="rowline">
    <Field label="Target">
      <SelectInput value={target} options={targets.map((t) => t.id)}
        onchange={(e: Event) => { target = (e.currentTarget as HTMLSelectElement).value; }} />
    </Field>
    {#if usablePresets.length > 0}
      <Field label="Function">
        <SelectInput value={preset?.id ?? ""}
          options={usablePresets.map((p) => ({ value: p.id, label: p.label }))}
          onchange={(e: Event) => pickPreset((e.currentTarget as HTMLSelectElement).value)} />
      </Field>
    {/if}
  </div>

  {#if preset && preset.fields.length > 0}
    <div class="rowline">
      {#each preset.fields as f (f.name)}
        <div class="paramcol">
          <Field label={f.label}>
            {#if f.type === "number"}
              <NumberInput value={Number(params[f.name] ?? f.default ?? 0)}
                oninput={(e: Event) => setParam(f.name, (e.currentTarget as HTMLInputElement).value)} />
            {:else}
              <TextInput mono value={params[f.name] ?? String(f.default ?? "")} placeholder={f.help ?? ""}
                oninput={(e: Event) => setParam(f.name, (e.currentTarget as HTMLInputElement).value)} />
            {/if}
          </Field>
          <div class="incline">
            <CheckboxInput label="count up" checked={!!inc[f.name]}
              onchange={(e: Event) => { inc = { ...inc, [f.name]: (e.currentTarget as HTMLInputElement).checked }; }} />
            {#if inc[f.name]}
              <span class="muted by">step</span>
              <NumberInput value={Number(step[f.name] ?? 1)}
                oninput={(e: Event) => { step = { ...step, [f.name]: (e.currentTarget as HTMLInputElement).value }; }} />
            {/if}
          </div>
        </div>
      {/each}
    </div>
  {:else if !preset}
    <div class="rowline">
      <Field label="Address" grow hint={'Use {i} (0-based) or {n} (1-based) as index placeholders'}>
        <TextInput mono value={addressTpl} placeholder={'/Page4/Key{n}'}
          oninput={(e: Event) => { addressTpl = (e.currentTarget as HTMLInputElement).value; }} />
      </Field>
      <Field label="Command (on press)" grow>
        <TextInput mono value={commandTpl} placeholder="e.g. Go Executor via /cmd"
          oninput={(e: Event) => { commandTpl = (e.currentTarget as HTMLInputElement).value; }} />
      </Field>
    </div>
  {/if}

  <div class="rowline">
    <Field label="Trigger">
      <SelectInput value={trigger} options={triggers}
        onchange={(e: Event) => { trigger = (e.currentTarget as HTMLSelectElement).value as Trigger; }} />
    </Field>
    {#if trigger === "hold"}
      <Field label="Hold (ms)">
        <NumberInput min={0} max={60000} value={holdMs}
          oninput={(e: Event) => { holdMs = Number((e.currentTarget as HTMLInputElement).value); }} />
      </Field>
    {/if}
    <Field label="Mode">
      <SelectInput value={mode} options={modes}
        onchange={(e: Event) => { mode = (e.currentTarget as HTMLSelectElement).value as Mode; }} />
    </Field>
    {#if mode === "toggle"}
      <Field label="LED color">
        <SelectInput value={ledColor} options={LED_COLORS}
          onchange={(e: Event) => { ledColor = (e.currentTarget as HTMLSelectElement).value; }} />
      </Field>
      <Field label="LED mode">
        <SelectInput value={ledMode} options={LED_MODES}
          onchange={(e: Event) => { ledMode = (e.currentTarget as HTMLSelectElement).value; }} />
      </Field>
    {/if}
  </div>

  {#if preview.length > 0}
    <pre class="preview">{preview.slice(0, PREVIEW_MAX).join("\n")}{generated.length > PREVIEW_MAX ? `\n… and ${generated.length - PREVIEW_MAX} more` : ""}</pre>
  {/if}

  <div class="rowline">
    <Button variant="primary" disabled={!source || !target || generated.length === 0} onclick={submit}>
      ⚡ Add {generated.length} binding{generated.length === 1 ? "" : "s"}
    </Button>
    {#if resultMsg}<span class="muted">{resultMsg}</span>{/if}
  </div>
</Card>

<style>
  .paramcol {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
  }
  .incline {
    display: flex;
    align-items: center;
    gap: 0.4rem;
  }
  .incline :global(.nin) {
    width: 4.5rem;
  }
  .by {
    font-size: 0.8rem;
  }
  .preview {
    background: #0c0f14;
    border: 1px solid var(--border);
    border-radius: 0.4rem;
    padding: 0.5rem 0.7rem;
    margin: 0.4rem 0 0.8rem;
    font-size: 0.8rem;
    color: var(--muted);
    max-height: 16rem;
    overflow: auto;
    white-space: pre;
  }
</style>
