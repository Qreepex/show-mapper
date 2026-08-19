<script lang="ts">
  import { onMount } from "svelte";
  import { api, ApiError } from "$lib/api";
  import type {
    Config,
    ControlKind,
    Meta,
    ProfileConfig,
    SourceConfig,
    TargetConfig,
  } from "$lib/types";

  let meta = $state<Meta | null>(null);
  let draft = $state<Config | null>(null);
  let saveMsg = $state<{ text: string; ok: boolean } | null>(null);
  let importEl = $state<HTMLInputElement>() as HTMLInputElement;

  async function reloadDraft() {
    draft = structuredClone(await api.config());
  }

  async function importFile(e: Event) {
    const file = (e.currentTarget as HTMLInputElement).files?.[0];
    saveMsg = null;
    if (!file) return;
    try {
      await api.importConfig(file);
      saveMsg = { text: "Config imported & applied.", ok: true };
      await reloadDraft();
    } catch (err) {
      saveMsg = { text: err instanceof ApiError ? err.errors.join("\n") : String(err), ok: false };
    } finally {
      if (importEl) importEl.value = ""; // allow re-selecting the same file
    }
  }

  onMount(async () => {
    const [m, c] = await Promise.all([api.meta(), api.config()]);
    meta = m;
    draft = structuredClone(c);
  });

  // ---- generic helpers -------------------------------------------------

  function num(v: unknown): number | undefined {
    return typeof v === "number" ? v : typeof v === "string" && v !== "" ? Number(v) : undefined;
  }
  function str(v: unknown): string {
    return typeof v === "string" ? v : "";
  }

  function setOpt(obj: { options?: Record<string, unknown> }, name: string, value: unknown) {
    obj.options = { ...(obj.options ?? {}), [name]: value };
  }

  // ---- sources / targets ------------------------------------------------

  function addSource() {
    if (!draft || !meta) return;
    const t = meta.sourceTypes[0];
    draft.sources.push({ id: `source-${draft.sources.length + 1}`, type: t?.type ?? "midi", options: {} });
  }
  function addTarget() {
    if (!draft || !meta) return;
    draft.targets.push({ id: `target-${draft.targets.length + 1}`, type: meta.targetTypes[0]?.type ?? "osc", options: {} });
  }

  function profileChoices(sourceType: string): { id: string; name: string }[] {
    if (!meta || !draft) return [];
    const builtin =
      meta.sourceTypes.find((t) => t.type === sourceType)?.profiles?.map((p) => ({ id: p.id, name: p.name })) ?? [];
    const custom = (draft.profiles ?? [])
      .filter((p) => p.type === sourceType)
      .map((p) => ({ id: p.id, name: p.name || p.id + " (custom)" }));
    return [...builtin, ...custom];
  }

  // ---- custom profiles (boards) -----------------------------------------

  function newProfile(): ProfileConfig {
    return {
      id: `board-${(draft?.profiles?.length ?? 0) + 1}`,
      type: "midi",
      name: "",
      match: [],
      led: { style: "onOff" },
      controls: [],
    };
  }

  function addControl(p: ProfileConfig) {
    p.controls.push({ id: `ctrl-${p.controls.length + 1}`, kind: "button", label: "", hasLED: false });
  }

  function setNum(c: { note?: number; cc?: number }, field: "note" | "cc", raw: string) {
    c[field] = raw === "" ? undefined : Math.trunc(Number(raw));
  }
  function csvToList(v: string): string[] {
    return v.split(",").map((s) => s.trim()).filter(Boolean);
  }

  async function save() {
    if (!draft) return;
    saveMsg = null;
    if (draft.updates && !draft.updates.repo?.trim()) {
      draft.updates = undefined; // empty repo string = feature off
    }
    try {
      await api.saveConfig(draft);
      saveMsg = { text: "Saved — connectors reloaded with the new settings.", ok: true };
    } catch (e) {
      saveMsg = { text: e instanceof ApiError ? e.errors.join("\n") : String(e), ok: false };
    }
  }
</script>

<h1>Settings</h1>

{#if !draft || !meta}
  <p class="muted">Loading…</p>
{:else}
  <div class="row" style="margin-bottom:1rem">
    <button class="primary" onclick={save}>Save &amp; apply</button>
    {#if saveMsg}
      <span class={saveMsg.ok ? "flash-ok" : "flash-err"}>{saveMsg.text}</span>
    {/if}
  </div>

  <div class="card">
    <h2>Config file</h2>
    <div class="row">
      <a class="btnlike" href={api.exportConfigURL} download="showbridge.yaml">
        ⭳ Download config (YAML)
      </a>
      <button onclick={() => importEl.click()}>⭱ Import config (YAML)…</button>
      <input bind:this={importEl} type="file" accept=".yaml,.yml" hidden onchange={importFile} />
      <span class="muted">Move configs between machines; edits also hot-reload from disk.</span>
    </div>
  </div>

  <div class="card">
    <h2>HTTP / Web UI</h2>
    <label for="listen">Listen address (localhost-only by default; use 0.0.0.0:8080 on trusted show networks)</label>
    <input id="listen" class="mono" bind:value={draft.http.listen} />
  </div>

  <div class="card">
    <h2>Software updates</h2>
    <div class="row">
      <div class="grow">
        <label for="updrepo">GitHub repo for releases (&quot;owner/name&quot;)</label>
        <input id="updrepo" class="mono" style="width:100%" placeholder="yourname/showbridge"
          value={draft.updates?.repo ?? ""}
          oninput={(e) => {
            draft && (draft.updates = { repo: e.currentTarget.value, autoCheck: draft.updates?.autoCheck ?? false });
          }} />
      </div>
      <div class="row" style="gap:0.4rem; align-items:center">
        <input id="updauto" type="checkbox" checked={draft.updates?.autoCheck ?? false}
          onchange={(e) => {
            draft && (draft.updates = { repo: draft.updates?.repo ?? "", autoCheck: e.currentTarget.checked });
          }} />
        <label for="updauto" style="margin:0">Check automatically on startup</label>
      </div>
    </div>
    <p class="muted hint">
      Off when repo is empty. Downloads are checksum-verified against the release's
      <code>checksums.txt</code>; applying swaps the binary in place (restart to run it).
    </p>
  </div>

  <div class="card">
    <h2>Sources</h2>
    {#each draft.sources as s, i (i)}
      <div class="inst">
        <div class="row">
          <div>
            <label for={"sid-" + i}>ID</label>
            <input id={"sid-" + i} class="mono" bind:value={s.id} />
          </div>
          <div>
            <label for={"stype-" + i}>Type</label>
            <select id={"stype-" + i} value={s.type}
              onchange={(e) => { s.type = e.currentTarget.value; s.profile = undefined; }}>
              {#each meta.sourceTypes as t (t.type)}
                <option value={t.type}>{t.type}</option>
              {/each}
            </select>
          </div>
          <div>
            <label for={"sprof-" + i}>Board profile</label>
            <select id={"sprof-" + i} value={s.profile ?? ""}
              onchange={(e) => { s.profile = e.currentTarget.value || undefined; }}>
              <option value="">(auto-detect from device name)</option>
              {#each profileChoices(s.type) as p (p.id)}
                <option value={p.id}>{p.id} — {p.name}</option>
              {/each}
            </select>
          </div>
          <button class="danger" onclick={() => draft && draft.sources.splice(i, 1)}>✕</button>
        </div>
        <div class="row opts">
          {#each meta.sourceTypes.find((t) => t.type === s.type)?.options ?? [] as f (f.name)}
            <div>
              <label for={"s-" + i + "-" + f.name}>{f.label}</label>
              {#if f.type === "number"}
                <input id={"s-" + i + "-" + f.name} type="number"
                  value={num(s.options?.[f.name]) ?? num(f.default) ?? ""}
                  oninput={(e) => setOpt(s, f.name, Number(e.currentTarget.value))} />
              {:else}
                <input id={"s-" + i + "-" + f.name}
                  value={str(s.options?.[f.name]) || str(f.default)}
                  placeholder={f.help ?? ""}
                  oninput={(e) => setOpt(s, f.name, e.currentTarget.value)} />
              {/if}
            </div>
          {/each}
        </div>
        {#if s.type === "midi"}
          <p class="muted hint">
            Tip: <code>device</code> is a case-insensitive substring of the OS MIDI port name
            (see Dashboard →“List MIDI ports”). Unknown buttons show up as
            <code>note:NN</code>/<code>cc:NN</code> in the Dashboard ticker for easy discovery.
          </p>
        {/if}
      </div>
    {/each}
    <button onclick={addSource}>+ Add source</button>
  </div>

  <div class="card">
    <h2>Targets</h2>
    {#each draft.targets as t, i (i)}
      <div class="inst">
        <div class="row">
          <div>
            <label for={"tid-" + i}>ID</label>
            <input id={"tid-" + i} class="mono" bind:value={t.id} />
          </div>
          <div>
            <label for={"ttype-" + i}>Type</label>
            <select id={"ttype-" + i} value={t.type}
              onchange={(e) => { t.type = e.currentTarget.value; }}>
              {#each meta.targetTypes as ty (ty.type)}
                <option value={ty.type}>{ty.type}</option>
              {/each}
            </select>
          </div>
          <button class="danger" onclick={() => draft && draft.targets.splice(i, 1)}>✕</button>
        </div>
        <div class="row opts">
          {#each meta.targetTypes.find((ty) => ty.type === t.type)?.options ?? [] as f (f.name)}
            <div>
              <label for={"t-" + i + "-" + f.name}>{f.label}</label>
              {#if f.type === "number"}
                <input id={"t-" + i + "-" + f.name} type="number"
                  value={num(t.options?.[f.name]) ?? num(f.default) ?? ""}
                  oninput={(e) => setOpt(t, f.name, Number(e.currentTarget.value))} />
              {:else}
                <input id={"t-" + i + "-" + f.name}
                  value={str(t.options?.[f.name]) || str(f.default)}
                  placeholder={f.help ?? ""}
                  oninput={(e) => setOpt(t, f.name, e.currentTarget.value)} />
              {/if}
            </div>
          {/each}
        </div>
      </div>
    {/each}
    <button onclick={addTarget}>+ Add target</button>
  </div>

  <div class="card">
    <h2>Custom boards (user-defined profiles)</h2>
    <p class="muted">
      Built-in boards (e.g. <code>apc-mini-mk2</code>) ship with the app. Any other
      controller can be described here and used via <code>profile</code> in a source.
      Discover note/CC numbers on the Dashboard (press a button → <code>note:NN</code>)
      or with <code>showbridge midi monitor</code>.
    </p>
    {#each draft.profiles ?? [] as p, pi (pi)}
      <div class="inst">
        <div class="row">
          <div>
            <label for={"pid-" + pi}>ID</label>
            <input id={"pid-" + pi} class="mono" bind:value={p.id} />
          </div>
          <div>
            <label for={"pname-" + pi}>Name</label>
            <input id={"pname-" + pi} bind:value={p.name} />
          </div>
          <div class="grow">
            <label for={"pmatch-" + pi}>Match (comma-separated port name substrings)</label>
            <input id={"pmatch-" + pi} style="width:100%" value={(p.match ?? []).join(", ")}
              oninput={(e) => { p.match = csvToList(e.currentTarget.value); }} />
          </div>
          <div>
            <label for={"pled-" + pi}>LED style</label>
            <select id={"pled-" + pi} value={p.led?.style ?? "none"}
              onchange={(e) => { p.led = { ...p.led, style: e.currentTarget.value }; }}>
              {#each meta.ledStyles as st (st)}
                <option value={st}>{st}</option>
              {/each}
            </select>
          </div>
          <button class="danger" onclick={() => draft && draft.profiles && draft.profiles.splice(pi, 1)}>✕</button>
        </div>

        <table class="ctrllist">
          <thead>
            <tr><th>ID</th><th>Label</th><th>Kind</th><th>Note (0–127)</th><th>CC (0–127)</th><th>LED</th><th></th></tr>
          </thead>
          <tbody>
            {#each p.controls as c, ci (ci)}
              <tr>
                <td><input class="mono" style="width:7rem" bind:value={c.id} /></td>
                <td><input style="width:9rem" bind:value={c.label} /></td>
                <td>
                  <select value={c.kind} onchange={(e) => { c.kind = e.currentTarget.value as ControlKind; }}>
                    {#each meta.controlKinds as k (k)}
                      <option value={k}>{k}</option>
                    {/each}
                  </select>
                </td>
                <td><input type="number" min="0" max="127" style="width:5rem" value={c.note ?? ""}
                  oninput={(e) => setNum(c, "note", e.currentTarget.value)} /></td>
                <td><input type="number" min="0" max="127" style="width:5rem" value={c.cc ?? ""}
                  oninput={(e) => setNum(c, "cc", e.currentTarget.value)} /></td>
                <td><input type="checkbox" checked={c.hasLED ?? false}
                  onchange={(e) => { c.hasLED = e.currentTarget.checked; }} /></td>
                <td><button class="danger" onclick={() => p.controls.splice(ci, 1)}>✕</button></td>
              </tr>
            {/each}
          </tbody>
        </table>
        <button onclick={() => addControl(p)}>+ Add control</button>
      </div>
    {/each}
    <button onclick={() => draft && (draft.profiles = [...(draft.profiles ?? []), newProfile()])}>
      + Add board
    </button>
  </div>
{/if}

<style>
  .card { margin-bottom: 1rem; }
  .inst {
    border: 1px solid var(--border);
    border-radius: 0.5rem;
    padding: 0.8rem;
    margin-bottom: 0.8rem;
    background: rgba(255, 255, 255, 0.015);
  }
  .opts { margin-top: 0.5rem; flex-wrap: wrap; }
  .hint { margin: 0.4rem 0 0; font-size: 0.85rem; }
  .ctrllist input, .ctrllist select { width: 100%; }
  code { background: var(--panel-2); padding: 0 0.3em; border-radius: 0.3em; }
  .btnlike {
    display: inline-block;
    background: var(--panel-2);
    color: var(--text);
    border: 1px solid var(--border);
    border-radius: 0.4rem;
    padding: 0.4rem 0.8rem;
    text-decoration: none;
  }
  .btnlike:hover { border-color: var(--accent); }
</style>
