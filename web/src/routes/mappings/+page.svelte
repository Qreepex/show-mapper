<script lang="ts">
  import { onMount } from "svelte";
  import { api, ApiError } from "$lib/api";
  import type {
    ActionType,
    Binding,
    Config,
    Meta,
    Mode,
    ProfileControl,
    Trigger,
  } from "$lib/types";

  let meta = $state<Meta | null>(null);
  let cfg = $state<Config | null>(null);
  let draft = $state<Binding[]>([]);
  let saveMsg = $state<{ text: string; ok: boolean } | null>(null);

  onMount(async () => {
    [meta, cfg] = await Promise.all([api.meta(), api.config()]);
    draft = structuredClone(cfg.bindings ?? []);
  });

  // All controls a source offers = its profile's controls (built-in or custom).
  function controlsFor(sourceId: string): { id: string; label: string }[] {
    if (!cfg || !meta) return [];
    const src = cfg.sources.find((s) => s.id === sourceId);
    if (!src) return [];
    const builtin = meta.sourceTypes
      .find((t) => t.type === src.type)
      ?.profiles?.find((p) => p.id === src.profile);
    if (builtin) return builtin.controls.map((c) => ({ id: c.id, label: c.label }));
    const custom = (cfg.profiles ?? [])
      .filter((p) => p.type === src.type && p.id === src.profile)
      .flatMap((p) => p.controls as ProfileControl[]);
    return custom.map((c) => ({ id: c.id, label: c.label }));
  }

  function addBinding() {
    if (!cfg) return;
    draft.push({
      source: cfg.sources[0]?.id ?? "",
      control: "",
      trigger: "pressed",
      mode: "momentary",
      target: cfg.targets[0]?.id ?? "",
      action: { type: "command", address: "/cmd", command: "", valueType: "int" },
      led: { color: "green", mode: "on" },
    });
  }

  function removeBinding(i: number) {
    draft.splice(i, 1);
  }

  async function save() {
    if (!cfg) return;
    saveMsg = null;
    try {
      await api.saveConfig({ ...cfg, bindings: draft });
      saveMsg = { text: "Saved — connectors reloaded.", ok: true };
      cfg = await api.config(); // backend may have normalized fields
      draft = structuredClone(cfg.bindings ?? []);
    } catch (e) {
      saveMsg = {
        text: e instanceof ApiError ? e.errors.join("\n") : String(e),
        ok: false,
      };
    }
  }
</script>

<h1>Mappings</h1>

{#if !cfg || !meta}
  <p class="muted">Loading…</p>
{:else}
  {#if cfg.sources.length === 0 || cfg.targets.length === 0}
    <div class="card">
      <p>
        You need at least one <a href="/settings">source and one target</a> before
        configuring mappings.
      </p>
    </div>
  {:else}
    <div class="row" style="margin-bottom: 1rem">
      <button class="primary" onclick={addBinding}>+ Add binding</button>
      <button onclick={save}>Save &amp; apply</button>
      {#if saveMsg}
        <span class={saveMsg.ok ? "flash-ok" : "flash-err"}>{saveMsg.text}</span>
      {/if}
    </div>

    {#if draft.length === 0}
      <p class="muted">No bindings yet. Tip: use the Dashboard ticker to find control IDs (like <code>pad-0-0</code> or <code>note:36</code>) while pressing buttons.</p>
    {/if}

    {#each draft as b, i (i)}
      <div class="card bind">
        <div class="rowline">
          <div>
            <label for={"src-" + i}>Source</label>
            <select id={"src-" + i} value={b.source} onchange={(e) => { b.source = e.currentTarget.value; }}>
              {#each cfg.sources as s (s.id)}
                <option value={s.id}>{s.id} ({s.type}{s.profile ? ":" + s.profile : ""})</option>
              {/each}
            </select>
          </div>
          <div>
            <label for={"ctrl-" + i}>Control</label>
            <input id={"ctrl-" + i} list={"ctrllist-" + i} placeholder="pad-0-0 …" value={b.control}
              oninput={(e) => { b.control = e.currentTarget.value; }} />
            <datalist id={"ctrllist-" + i}>
              {#each controlsFor(b.source) as c (c.id)}
                <option value={c.id}>{c.label}</option>
              {/each}
            </datalist>
          </div>
          <div>
            <label for={"trig-" + i}>Trigger</label>
            <select id={"trig-" + i} value={b.trigger}
              onchange={(e) => { b.trigger = e.currentTarget.value as Trigger; }}>
              {#each meta.triggers as t (t)}
                <option value={t}>{t}</option>
              {/each}
            </select>
          </div>
          {#if b.trigger === "hold"}
            <div>
              <label for={"hold-" + i}>Hold (ms)</label>
              <input id={"hold-" + i} type="number" min="0" max="60000" value={b.holdMs ?? 500}
                oninput={(e) => { b.holdMs = Number(e.currentTarget.value); }} />
            </div>
          {/if}
          <div>
            <label for={"mode-" + i}>Mode</label>
            <select id={"mode-" + i} value={b.mode ?? "momentary"}
              onchange={(e) => { b.mode = e.currentTarget.value as Mode; }}>
              {#each meta.modes as m (m)}
                <option value={m}>{m}</option>
              {/each}
            </select>
          </div>
          <div>
            <label for={"tgt-" + i}>Target</label>
            <select id={"tgt-" + i} value={b.target}
              onchange={(e) => { b.target = e.currentTarget.value; }}>
              {#each cfg.targets as t (t.id)}
                <option value={t.id}>{t.id} ({t.type})</option>
              {/each}
            </select>
          </div>
          <button class="danger del" title="Remove binding" onclick={() => removeBinding(i)}>✕</button>
        </div>

        <div class="rowline">
          <div>
            <label for={"atype-" + i}>Action</label>
            <select id={"atype-" + i} value={b.action.type}
              onchange={(e) => { b.action.type = e.currentTarget.value as ActionType; }}>
              {#each meta.actionTypes as a (a)}
                <option value={a}>{a}</option>
              {/each}
            </select>
          </div>
          <div class="grow">
            <label for={"addr-" + i}>Address (e.g. /cmd, /Page1/Fader201, /Page1/Key201)</label>
            <input id={"addr-" + i} class="grow mono" value={b.action.address} style="width:100%"
              oninput={(e) => { b.action.address = e.currentTarget.value; }} />
          </div>
          <div>
            <label for={"vt-" + i}>Value type</label>
            <select id={"vt-" + i} value={b.action.valueType ?? "int"}
              onchange={(e) => { b.action.valueType = e.currentTarget.value as "int" | "float"; }}>
              <option value="int">int</option>
              <option value="float">float</option>
            </select>
          </div>
        </div>

        {#if b.action.type === "command"}
          <div class="rowline">
            <div class="grow">
              <label for={"cmd-" + i}>Command (on press)</label>
              <input id={"cmd-" + i} class="mono" style="width:100%" placeholder="Go Executor 1.201"
                value={b.action.command ?? ""}
                oninput={(e) => { b.action.command = e.currentTarget.value; }} />
            </div>
            <div class="grow">
              <label for={"rcmd-" + i}>Command (on release, optional)</label>
              <input id={"rcmd-" + i} class="mono" style="width:100%"
                value={b.action.releaseCommand ?? ""}
                oninput={(e) => { b.action.releaseCommand = e.currentTarget.value; }} />
            </div>
          </div>
        {:else if b.action.type === "value"}
          <div class="rowline">
            <div>
              <label for={"pv-" + i}>Press value</label>
              <input id={"pv-" + i} type="number" step="any" value={b.action.pressValue ?? 1}
                oninput={(e) => { b.action.pressValue = Number(e.currentTarget.value); }} />
            </div>
            <div>
              <label for={"rv-" + i}>Release value</label>
              <input id={"rv-" + i} type="number" step="any" value={b.action.releaseValue ?? 0}
                oninput={(e) => { b.action.releaseValue = Number(e.currentTarget.value); }} />
            </div>
          </div>
        {:else if b.action.type === "fader"}
          <div class="rowline">
            <div>
              <label for={"rmin-" + i}>Range min</label>
              <input id={"rmin-" + i} type="number" step="any" value={b.action.range?.[0] ?? 0}
                oninput={(e) => { b.action.range = [Number(e.currentTarget.value), b.action.range?.[1] ?? 100]; }} />
            </div>
            <div>
              <label for={"rmax-" + i}>Range max</label>
              <input id={"rmax-" + i} type="number" step="any" value={b.action.range?.[1] ?? 100}
                oninput={(e) => { b.action.range = [b.action.range?.[0] ?? 0, Number(e.currentTarget.value)]; }} />
            </div>
          </div>
        {/if}

        {#if b.mode === "toggle"}
          <div class="rowline">
            <div>
              <label for={"ledc-" + i}>LED color (toggle)</label>
              <select id={"ledc-" + i} value={b.led?.color ?? "green"}
                onchange={(e) => { b.led = { ...b.led, color: e.currentTarget.value }; }}>
                {#each ["green", "red", "orange", "yellow", "cyan", "blue", "purple", "pink", "white"] as c (c)}
                  <option value={c}>{c}</option>
                {/each}
              </select>
            </div>
            <div>
              <label for={"ledm-" + i}>LED mode</label>
              <select id={"ledm-" + i} value={b.led?.mode ?? "on"}
                onchange={(e) => { b.led = { ...b.led, mode: e.currentTarget.value }; }}>
                {#each ["on", "blink", "pulse"] as m (m)}
                  <option value={m}>{m}</option>
                {/each}
              </select>
            </div>
          </div>
        {/if}
      </div>
    {/each}
  {/if}
{/if}

<style>
  .bind { margin-bottom: 0.8rem; }
  .rowline { display: flex; gap: 0.8rem; align-items: flex-end; margin-bottom: 0.6rem; flex-wrap: wrap; }
  .grow { flex: 1; min-width: 200px; }
  .del { margin-left: auto; }
</style>
