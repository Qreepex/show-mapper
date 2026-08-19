<script lang="ts">
  import type { Snippet } from "svelte";
  let { title, children }: { title: string; children?: Snippet } = $props();
</script>

<div class="pagehead">
  <h1>{title}</h1>
  <div class="actions">{@render children?.()}</div>
</div>

<style>
  .pagehead {
    display: flex;
    align-items: center;
    gap: 0.8rem;
    margin-bottom: 1rem;
    flex-wrap: wrap;
  }
  .pagehead h1 {
    margin: 0;
    font-size: 1.35rem;
  }
  .actions {
    display: flex;
    gap: 0.5rem;
    align-items: center;
    margin-left: auto;
  }
</style>
