<script lang="ts">
  import type { Snippet } from "svelte";
  let { children }: { children?: Snippet } = $props();
</script>

<div class="empty">{@render children?.()}</div>

<style>
  .empty {
    border: 1px dashed var(--border);
    border-radius: 0.6rem;
    padding: 2rem 1rem;
    text-align: center;
    color: var(--muted);
    display: flex;
    flex-direction: column;
    gap: 0.8rem;
    align-items: center;
    margin-bottom: 1rem;
  }
</style>
