<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "$lib/api";
  import { ConfigDraft } from "$lib/draft.svelte";
  import type { ActionPreset, ActionType, Binding, Mode, Trigger } from "$lib/types";
  import Card from "$lib/ui/Card.svelte";
  import PageHeader from "$lib/ui/PageHeader.svelte";
  import Button from "$lib/ui/Button.svelte";
  import LinkButton from "$lib/ui/LinkButton.svelte";
  import ImportButton from "$lib/ui/ImportButton.svelte";
  import SaveBar from "$lib/ui/SaveBar.svelte";
  import Field from "$lib/ui/Field.svelte";
  import TextInput from "$lib/ui/TextInput.svelte";
  import NumberInput from "$lib/ui/NumberInput.svelte";
  import SelectInput from "$lib/ui/SelectInput.svelte";
  import EmptyState from "$lib/ui/EmptyState.svelte";

  const d = new ConfigDraft();
  onMount(() => d.init());

  function bindings(): Binding[] {
    return d.cfg?.bindings ?? [];
  }

  function controlsFor(sourceId: string): { id: string; label: string }[] {
    if (!d.cfg || !d.meta) return [];
    const src = d.cfg.sources.find((s) => s.id === sourceId);
    if (!src) return [];
    const builtin = d.meta.sourceTypes
      .find((t) => t.type === src.type)
      ?.profiles?.find((p) => p.id === src.profile);
    if (builtin) return builtin.controls.map((c) => ({ id: c.id, label: c.label }));
    const customs = (d.cfg.profiles ?? [])
      .filter((p) => p.type === src.type && p.id === src.profile)
      .flatMap((p) => p.controls);
    return customs.map((c) => ({ id: c.id, label: c.label }));
  }

  // ---- target-typed actions -------------------------------------------------
  // Binding targets whose type has helper-module presets (e.g. gma3) get a
  // friendly form (action dropdown + parameter fields); everything else gets
  // the generic raw fields (command/value/fader).

  function targetTypeOf(b: Binding): string {
    return d.cfg?.targets.find((t) => t.id === b.target)?.type ?? "";
  }

  function presetsFor(targetType: string): ActionPreset[] {
    return (d.meta?.presets ?? []).filter(
      (p) => !p.targetTypes || p.targetTypes.length === 0 || p.targetTypes.includes(targetType),
    );
  }

  function presetById(id: string): ActionPreset | undefined {
    return d.meta?.presets.find((p) => p.id === id);
  }

  function defaultPresetParams(p: ActionPreset): Record<string, string> {
    const out: Record<string, string> = {};
    for (const f of p.fields) out[f.name] = String(f.default ?? "");
    return out;
  }

  /** Switch a binding between preset mode and generic/raw mode. */
  function setActionMode(b: Binding, presetId: string) {
    if (presetId === "") {
      b.action = {
        type: "command",
        address: b.action.address || "/cmd",
        command: b.action.command ?? "",
        valueType: "int",
      };
      return;
    }
    const p = presetById(presetId);
    b.action = {
      type: "preset",
      preset: presetId,
      params: p ? { ...defaultPresetParams(p) } : {},
      address: "",
      valueType: "int",
    };
  }

  function paramStr(b: Binding, name: string): string {
    const v = b.action.params?.[name];
    return v === undefined || v === null ? "" : String(v);
  }

  function addBinding() {
    if (!d.cfg) return;
    const tgt = d.cfg.targets[0]?.id ?? "";
    const tpres = presetsFor(targetTypeOf({ target: tgt } as Binding));
    const action =
      tpres.length > 0
        ? { type: "preset" as ActionType, preset: tpres[0].id, params: defaultPresetParams(tpres[0]), address: "", valueType: "int" as const }
        : { type: "command" as ActionType, address: "/cmd", command: "", valueType: "int" as const };
    d.cfg.bindings.push({
      source: d.cfg.sources[0]?.id ?? "",
      control: "",
      trigger: "pressed",
      mode: "momentary",
      target: tgt,
      action,
      led: { color: "green", mode: "on" },
    });
  }

  const LED_COLORS = ["green", "red", "orange", "yellow", "cyan", "blue", "purple", "pink", "white"];
  const LED_MODES = ["on", "blink", "pulse"];

  function numdef(v: unknown): number {
    return typeof v === "number" ? v : 0;
  }
</script>

<PageHeader title="Mappings">
  <LinkButton href={api.exportSectionURL("bindings")} download="show-mapper-bindings.yaml">⭳ Export</LinkButton>
  <ImportButton onfile={(f) => d.importSection(f)} />
</PageHeader>

{#if !d.cfg || !d.meta}
  <p class="muted">Loading…</p>
{:else}
  <SaveBar onsave={() => d.save()} msg={d.msg}>
    <Button onclick={addBinding}>+ Add binding</Button>
  </SaveBar>

  {#if d.cfg.sources.length === 0 || d.cfg.targets.length === 0}
    <EmptyState>
      <p>
        You need at least one <a href="/sources">source</a> and one
        <a href="/targets">target</a> first.
      </p>
    </EmptyState>
  {:else if bindings().length === 0}
    <EmptyState>
      <p>No bindings yet. Press buttons on your board (or the <a href="/surface">Surface</a>) while watching the Dashboard ticker to discover control IDs (<code>pad-0-0</code>, <code>note:36</code>…).</p>
      <Button variant="primary" onclick={addBinding}>+ Add your first binding</Button>
    </EmptyState>
  {/if}

  {#each bindings() as b, i (i)}
    {@const tpres = presetsFor(targetTypeOf(b))}
    <Card title={`${b.source}:${b.control || "?"} → ${b.target}`}>
      {#snippet actions()}
        <Button variant="danger" onclick={() => d.cfg && d.cfg.bindings.splice(i, 1)}>✕ remove</Button>
      {/snippet}

      <div class="rowline">
        <Field label="Source">
          <SelectInput value={b.source} options={d.cfg.sources.map((s) => s.id)}
            onchange={(e: Event) => { b.source = (e.currentTarget as HTMLSelectElement).value; }} />
        </Field>
        <Field label="Control (pick or type raw)">
          <TextInput mono list={`ctrllist-${i}`} value={b.control} placeholder="pad-0-0 …"
            oninput={(e: Event) => { b.control = (e.currentTarget as HTMLInputElement).value; }} />
          <datalist id={`ctrllist-${i}`}>
            {#each controlsFor(b.source) as c (c.id)}
              <option value={c.id}>{c.label}</option>
            {/each}
          </datalist>
        </Field>
        <Field label="Trigger">
          <SelectInput value={b.trigger} options={d.meta.triggers}
            onchange={(e: Event) => { b.trigger = (e.currentTarget as HTMLSelectElement).value as Trigger; }} />
        </Field>
        {#if b.trigger === "hold"}
          <Field label="Hold (ms)">
            <NumberInput min={0} max={60000} value={b.holdMs ?? 500}
              oninput={(e: Event) => { b.holdMs = Number((e.currentTarget as HTMLInputElement).value); }} />
          </Field>
        {/if}
        <Field label="Mode">
          <SelectInput value={b.mode ?? "momentary"} options={d.meta.modes}
            onchange={(e: Event) => { b.mode = (e.currentTarget as HTMLSelectElement).value as Mode; }} />
        </Field>
        <Field label="Target">
          <SelectInput value={b.target} options={d.cfg.targets.map((t) => t.id)}
            onchange={(e: Event) => { b.target = (e.currentTarget as HTMLSelectElement).value; }} />
        </Field>
      </div>

      {#if tpres.length > 0}
        <!-- target with helper actions (e.g. grandMA3): ONLY the functions form -->
        <div class="rowline">
          <Field label="Action (grandMA3)">
            <SelectInput
              value={b.action.type === "preset" ? b.action.preset ?? "" : ""}
              options={tpres.map((p) => ({ value: p.id, label: p.label }))}
              allowEmpty="— pick a function —"
              onchange={(e: Event) => setActionMode(b, (e.currentTarget as HTMLSelectElement).value)}
            />
          </Field>
          {#if b.action.type === "preset" && b.action.preset}
            {#each presetById(b.action.preset)?.fields ?? [] as f (f.name)}
              <Field label={f.label} hint={f.help}>
                {#if f.type === "number"}
                  <NumberInput
                    value={Number(paramStr(b, f.name)) || numdef(f.default)}
                    oninput={(e: Event) => {
                      b.action.params = { ...(b.action.params ?? {}), [f.name]: (e.currentTarget as HTMLInputElement).value };
                    }} />
                {:else}
                  <TextInput
                    value={paramStr(b, f.name)}
                    placeholder={f.help ?? ""}
                    oninput={(e: Event) => {
                      b.action.params = { ...(b.action.params ?? {}), [f.name]: (e.currentTarget as HTMLInputElement).value };
                    }} />
                {/if}
              </Field>
            {/each}
          {/if}
        </div>
        {#if b.action.type === "preset" && presetById(b.action.preset ?? "")?.help}
          <p class="muted">{presetById(b.action.preset ?? "")?.help}</p>
        {:else if b.action.type !== "preset"}
          <p class="muted">
            This binding uses a raw action against a grandMA3 target — pick a grandMA3 function above.
          </p>
        {/if}
      {/if}

      {#if tpres.length === 0}
        <!-- generic / raw action form (targets without helper functions) -->
        <div class="rowline">
          <Field label="Action type">
            <SelectInput value={b.action.type} options={d.meta.actionTypes.filter((t) => t !== "preset")}
              onchange={(e: Event) => { b.action.type = (e.currentTarget as HTMLSelectElement).value as ActionType; }} />
          </Field>
          <Field label="Address" grow>
            <TextInput mono value={b.action.address} placeholder="/cmd, /Page1/Fader201, …"
              oninput={(e: Event) => { b.action.address = (e.currentTarget as HTMLInputElement).value; }} />
          </Field>
          <Field label="Value type">
            <SelectInput value={b.action.valueType ?? "int"} options={["int", "float"]}
              onchange={(e: Event) => { b.action.valueType = (e.currentTarget as HTMLSelectElement).value as "int" | "float"; }} />
          </Field>
        </div>

        {#if b.action.type === "command"}
          <div class="rowline">
            <Field label="Command (on press)" grow>
              <TextInput mono value={b.action.command ?? ""} placeholder="Go Executor 1.201"
                oninput={(e: Event) => { b.action.command = (e.currentTarget as HTMLInputElement).value; }} />
            </Field>
            <Field label="Command (on release, optional)" grow>
              <TextInput mono value={b.action.releaseCommand ?? ""}
                oninput={(e: Event) => { b.action.releaseCommand = (e.currentTarget as HTMLInputElement).value; }} />
            </Field>
          </div>
        {:else if b.action.type === "value"}
          <div class="rowline">
            <Field label="Press value">
              <NumberInput value={b.action.pressValue ?? 1}
                oninput={(e: Event) => { b.action.pressValue = Number((e.currentTarget as HTMLInputElement).value); }} />
            </Field>
            <Field label="Release value">
              <NumberInput value={b.action.releaseValue ?? 0}
                oninput={(e: Event) => { b.action.releaseValue = Number((e.currentTarget as HTMLInputElement).value); }} />
            </Field>
          </div>
        {:else if b.action.type === "fader"}
          <div class="rowline">
            <Field label="Range min">
              <NumberInput value={b.action.range?.[0] ?? 0}
                oninput={(e: Event) => { b.action.range = [Number((e.currentTarget as HTMLInputElement).value), b.action.range?.[1] ?? 100]; }} />
            </Field>
            <Field label="Range max">
              <NumberInput value={b.action.range?.[1] ?? 100}
                oninput={(e: Event) => { b.action.range = [b.action.range?.[0] ?? 0, Number((e.currentTarget as HTMLInputElement).value)]; }} />
            </Field>
          </div>
        {/if}
      {/if}
      {#if b.mode === "toggle"}
        <div class="rowline">
          <Field label="LED color (toggle)">
            <SelectInput value={b.led?.color ?? "green"} options={LED_COLORS}
              onchange={(e: Event) => { b.led = { ...b.led, color: (e.currentTarget as HTMLSelectElement).value }; }} />
          </Field>
          <Field label="LED mode">
            <SelectInput value={b.led?.mode ?? "on"} options={LED_MODES}
              onchange={(e: Event) => { b.led = { ...b.led, mode: (e.currentTarget as HTMLSelectElement).value }; }} />
          </Field>
        </div>
      {/if}
    </Card>
  {/each}
{/if}
