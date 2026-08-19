<script lang="ts">
  import type { Control } from "$lib/types";
  import { live } from "$lib/ws.svelte";

  // Renders one virtual surface instance as a playable board
  // (8×8 pads, buttons row, fader bank) and injects events via WS.
  let { sourceId, controls }: { sourceId: string; controls: Control[] } = $props();

  const pads = $derived(controls.filter((c) => c.kind === "pad"));
  const buttons = $derived(controls.filter((c) => c.kind === "button"));
  const faders = $derived(controls.filter((c) => c.kind === "fader"));

  let faderVal = $state<Record<string, number>>({});

  function press(control: string) {
    live.send("client.sim", { source: sourceId, control, kind: "pressed", value: 1 });
  }
  function release(control: string) {
    live.send("client.sim", { source: sourceId, control, kind: "released", value: 0 });
  }
  function setFader(control: string, v: number) {
    faderVal[control] = v;
    live.send("client.sim", { source: sourceId, control, kind: "value", value: v });
  }

  const rows = $derived([...new Set(pads.map((p) => p.row ?? 0))].sort((a, b) => b - a));
</script>

<div class="board">
  <div class="pads">
    {#each rows as row (row)}
      <div class="padrow">
        {#each pads.filter((c) => (c.row ?? 0) === row) as p (p.id)}
          <button
            class="pad"
            title={p.label}
            onmousedown={() => press(p.id)}
            onmouseup={() => release(p.id)}
            onmouseleave={(e: MouseEvent) => e.buttons && release(p.id)}
          >
            {(p.col ?? 0) + 1}
          </button>
        {/each}
      </div>
    {/each}
  </div>

  {#if buttons.length > 0}
    <div class="row">
      {#each buttons as b (b.id)}
        <button class="sbtn" onmousedown={() => press(b.id)} onmouseup={() => release(b.id)}>
          {b.label}
        </button>
      {/each}
    </div>
  {/if}

  {#if faders.length > 0}
    <div class="faders">
      {#each faders as f (f.id)}
        <div class="fader">
          <input
            class="fslider"
            type="range"
            min="0"
            max="1"
            step="0.01"
            value={faderVal[f.id] ?? 0}
            oninput={(e: Event) => setFader(f.id, Number((e.currentTarget as HTMLInputElement).value))}
          />
          <span class="flabel">{f.label}</span>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .board {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  .pads {
    display: inline-flex;
    flex-direction: column;
    gap: 6px;
  }
  .padrow {
    display: flex;
    gap: 6px;
  }
  .pad {
    width: 44px;
    height: 44px;
    border: 1px solid var(--border);
    border-radius: 6px;
    background: var(--panel-2);
    color: var(--muted);
    cursor: pointer;
    font-size: 0.75rem;
  }
  .pad:active {
    background: var(--accent);
    color: #06202e;
  }
  .sbtn {
    border: 1px solid var(--border);
    border-radius: 0.4rem;
    background: var(--panel-2);
    color: var(--text);
    padding: 0.35rem 0.7rem;
    cursor: pointer;
  }
  .sbtn:active {
    background: var(--accent);
    color: #06202e;
  }
  .faders {
    display: flex;
    gap: 1rem;
    flex-wrap: wrap;
  }
  .fader {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.3rem;
  }
  .fslider {
    writing-mode: vertical-lr;
    direction: rtl;
    height: 120px;
  }
  .flabel {
    font-size: 0.75rem;
    color: var(--muted);
  }
</style>
