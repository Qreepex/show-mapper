<script lang="ts">
  import { onMount } from "svelte";
  import { live } from "$lib/ws.svelte";
  import { api } from "$lib/api";
  import type { Config } from "$lib/types";
  import Card from "$lib/ui/Card.svelte";
  import PageHeader from "$lib/ui/PageHeader.svelte";
  import Button from "$lib/ui/Button.svelte";
  import EmptyState from "$lib/ui/EmptyState.svelte";
  import Msg from "$lib/ui/Msg.svelte";
  import SurfaceBoard from "$lib/ui/SurfaceBoard.svelte";

  // The virtual surface: play a `sim` source right here in the browser.
  // No hardware, no CGO build — events flow the real pipeline (see ticker).
  let cfg = $state<Config | null>(null);
  let msg = $state<{ text: string; ok: boolean } | null>(null);

  const simSources = $derived((cfg?.sources ?? []).filter((s) => s.type === "sim"));

  onMount(async () => {
    cfg = await api.config();
  });

  async function addSimSource() {
    msg = null;
    try {
      const c = await api.config();
      c.sources.push({ id: "sim", type: "sim" });
      await api.saveConfig(c);
      cfg = await api.config();
      msg = { text: "Virtual surface added.", ok: true };
    } catch (e) {
      msg = { text: String(e), ok: false };
    }
  }
</script>

<PageHeader title="Surface (virtual board)" />

{#if !cfg}
  <p class="muted">Loading…</p>
{:else if simSources.length === 0}
  <EmptyState>
    <p>
      No virtual surface configured. Add a <code>sim</code> source — a full 8×8 +
      buttons + faders board, right here in your browser. No hardware, no CGO build needed.
    </p>
    <Button variant="primary" onclick={addSimSource}>+ Add virtual surface</Button>
  </EmptyState>
{:else}
  {#each simSources as src (src.id)}
    <Card title={"Surface: " + src.id}>
      {@const controls = live.connector(src.id)?.controls ?? []}
      {#if controls.length === 0}
        <p class="muted">Waiting for the source to report its controls… (connectors reload on save)</p>
      {:else}
        <SurfaceBoard sourceId={src.id} {controls} />
        <p class="muted">Watch the action flow on the <a href="/">Dashboard</a> ticker.</p>
      {/if}
    </Card>
  {/each}
{/if}

<Msg {msg} />
