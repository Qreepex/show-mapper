<script lang="ts">
  import { live } from "$lib/ws.svelte";
  import { api } from "$lib/api";
  import type { ConnectorState } from "$lib/types";

  let inspectResult = $state<string | null>(null);

  async function listMidiPorts() {
    inspectResult = "…";
    try {
      const res = await api.inspectSource("midi");
      if (!res.ok) {
        inspectResult = `MIDI unavailable: ${res.error}`;
        return;
      }
      const ins = res.result?.in ?? [];
      const outs = res.result?.out ?? [];
      inspectResult =
        "MIDI inputs:\n" +
        (ins.length ? ins.map((p) => `  [${p.number}] ${p.name}`).join("\n") : "  (none)") +
        "\nMIDI outputs:\n" +
        (outs.length ? outs.map((p) => `  [${p.number}] ${p.name}`).join("\n") : "  (none)");
    } catch (e) {
      inspectResult = String(e);
    }
  }

  function stateClass(c: ConnectorState): string {
    return c.status.state === "connected"
      ? "connected"
      : c.status.state === "error"
        ? "error"
        : c.status.state === "connecting"
          ? "connecting"
          : "disconnected";
  }
</script>

<h1>Dashboard</h1>

<div class="grid two">
  <div class="card">
    <h2>Connectors</h2>
    {#if live.connectors.length === 0}
      <p class="muted">Waiting for backend state…</p>
    {:else}
      <table>
        <thead>
          <tr><th></th><th>ID</th><th>Kind</th><th>Type</th><th>Status</th></tr>
        </thead>
        <tbody>
          {#each live.connectors as c (c.kind + c.id)}
            <tr>
              <td><span class="dot {stateClass(c)}"></span></td>
              <td class="mono">{c.id}</td>
              <td class="muted">{c.kind}</td>
              <td class="muted">{c.type}</td>
              <td>
                <span class="mono">{c.status.state}</span>
                {#if c.status.detail}
                  <br /><span class="muted">{c.status.detail}</span>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>

  <div class="card">
    <h2>MIDI discovery</h2>
    <p class="muted">
      Lists OS MIDI ports as seen by this build. Use a unique substring as
      <code>device</code> in a source's options. (Needs a CGO build — release binaries are.)
    </p>
    <button onclick={listMidiPorts}>List MIDI ports</button>
    {#if inspectResult}
      <pre class="mono">{inspectResult}</pre>
    {/if}
  </div>
</div>

<div class="card" style="margin-top:1rem">
  <h2>Live activity</h2>
  {#if live.ticker.length === 0}
    <p class="muted">No events yet — press a button on your board.</p>
  {:else}
    <table>
      <tbody>
        {#each live.ticker.slice(0, 60) as e, i (e.ts.getTime() + ":" + i)}
          <tr>
            <td class="muted mono">{e.ts.toLocaleTimeString()}</td>
            <td>
              {#if e.kind === "target.action"}
                <span style:color={e.ok ? "var(--ok)" : "var(--err)"}>➜</span>
              {:else if e.kind === "source.event"}
                <span style:color="var(--accent)">◉</span>
              {:else}
                <span style:color="var(--muted)">⚙</span>
              {/if}
            </td>
            <td class="mono">{e.text}</td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</div>

<style>
  .grid.two { grid-template-columns: 1fr 1fr; }
  code { background: var(--panel-2); padding: 0 0.3em; border-radius: 0.3em; }
  pre { white-space: pre-wrap; }
</style>
