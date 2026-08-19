<script lang="ts">
  import { onMount } from "svelte";
  import { live } from "$lib/ws.svelte";
  import { api, ApiError } from "$lib/api";
  import Card from "$lib/ui/Card.svelte";
  import Button from "$lib/ui/Button.svelte";
  import StatusDot from "$lib/ui/StatusDot.svelte";
  import Msg from "$lib/ui/Msg.svelte";
  import PageHeader from "$lib/ui/PageHeader.svelte";

  let inspectResult = $state<string | null>(null);
  let updateBusy = $state(false);
  let updateMsg = $state<{ text: string; ok: boolean } | null>(null);

  onMount(async () => {
    try {
      live.update = await api.updateStatus();
    } catch {
      /* updater not configured — fine */
    }
  });

  async function listMidiPorts() {
    inspectResult = "…";
    try {
      const res = await api.inspectSource("midi");
      if (!res.ok) {
        inspectResult = `MIDI unavailable: ${res.error}
Tip: use the built-in virtual Surface tab while developing without a CGO build/board.`;
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

  async function checkUpdate() {
    updateBusy = true;
    updateMsg = null;
    try {
      live.update = await api.updateCheck();
      updateMsg = live.update.available
        ? { text: `v${live.update.latestVersion} available.`, ok: true }
        : { text: live.update.error || "You're up to date.", ok: true };
    } catch (e) {
      updateMsg = { text: e instanceof ApiError ? e.errors.join("; ") : String(e), ok: false };
    } finally {
      updateBusy = false;
    }
  }

  async function applyUpdate() {
    updateBusy = true;
    updateMsg = null;
    try {
      const r = await api.updateApply();
      updateMsg = { text: r.message, ok: true };
    } catch (e) {
      updateMsg = { text: e instanceof ApiError ? e.errors.join("; ") : String(e), ok: false };
    } finally {
      updateBusy = false;
    }
  }
</script>

<PageHeader title="Dashboard" />

{#if live.update?.available}
  <Card>
    <div class="row">
      <strong>⬆ Update available: v{live.update.latestVersion}</strong>
      <span class="muted">(running {live.update.current})</span>
      <Button variant="primary" onclick={applyUpdate} disabled={updateBusy}>
        Download &amp; install
      </Button>
      {#if live.update.latestURL}
        <a href={live.update.latestURL} target="_blank" rel="noreferrer">release notes</a>
      {/if}
      <Msg msg={updateMsg} />
    </div>
  </Card>
{/if}

<div class="grid-two">
  <Card title="Connectors">
    {#if live.connectors.length === 0}
      <p class="muted">
        No connectors configured yet — add some under
        <a href="/sources">Sources</a> / <a href="/targets">Targets</a>.
        Tip for development without hardware: add a <code>sim</code> source and play on the
        <a href="/surface">Surface</a> tab.
      </p>
    {:else}
      <table>
        <thead>
          <tr><th></th><th>ID</th><th>Kind</th><th>Type</th><th>Status</th></tr>
        </thead>
        <tbody>
          {#each live.connectors as c (c.kind + c.id)}
            <tr>
              <td><StatusDot state={c.status.state} /></td>
              <td class="mono">{c.id}</td>
              <td class="muted">{c.kind}</td>
              <td class="muted">{c.type}</td>
              <td>
                <span class="mono">{c.status.state}</span>
                {#if c.status.detail}<br /><span class="muted">{c.status.detail}</span>{/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </Card>

  <Card title="MIDI discovery">
    <p class="muted">
      Lists OS MIDI ports as seen by this build. Use a unique substring as
      <code>device</code> in a source's options. (Needs a CGO build — release binaries have it;
      during development without one, use the <a href="/surface">Surface</a> tab.)
    </p>
    <Button onclick={listMidiPorts}>List MIDI ports</Button>
    {#if inspectResult}
      <pre class="mono">{inspectResult}</pre>
    {/if}
  </Card>
</div>

<Card title="Software update">
  <div class="row">
    <Button onclick={checkUpdate} disabled={updateBusy}>Check now</Button>
    <Msg msg={updateMsg} />
    {#if live.update && !live.update.configured && !updateMsg}
      <span class="muted">
        Not configured — set <code>updates.repo</code> (&quot;owner/repo&quot;) in
        <a href="/settings">Settings</a>.
      </span>
    {/if}
  </div>
</Card>

<Card title="Live activity">
  {#if live.ticker.length === 0}
    <p class="muted">
      No events yet — press a button on your board or on the
      <a href="/surface">Surface</a> tab.
    </p>
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
</Card>
