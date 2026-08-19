<script lang="ts">
  import type { Snippet } from "svelte";
  let {
    title,
    actions,
    children,
  }: {
    title?: string;
    actions?: Snippet;
    children?: Snippet;
  } = $props();
</script>

<section class="card">
  {#if title || actions}
    <header class="head">
      {#if title}<h2>{title}</h2>{/if}
      {#if actions}<div class="actions">{@render actions()}</div>{/if}
    </header>
  {/if}
  <div class="body">{@render children?.()}</div>
</section>

<style>
  .card {
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 0.6rem;
    padding: 1rem;
    margin-bottom: 1rem;
  }
  .head {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    margin-bottom: 0.6rem;
  }
  .head h2 {
    margin: 0;
    font-size: 0.8rem;
    color: var(--muted);
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }
  .actions {
    margin-left: auto;
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }
</style>
