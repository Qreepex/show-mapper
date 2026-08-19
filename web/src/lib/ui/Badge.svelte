<script lang="ts">
  import type { Snippet } from "svelte";
  let {
    variant = "muted",
    children,
  }: {
    variant?: "ok" | "err" | "warn" | "muted";
    children?: Snippet;
  } = $props();
</script>

<span class="badge {variant}">{@render children?.()}</span>

<style>
  .badge {
    display: inline-block;
    padding: 0.1rem 0.55rem;
    border-radius: 999px;
    font-size: 0.75rem;
    border: 1px solid var(--border);
  }
  .ok {
    color: var(--ok);
    border-color: var(--ok);
  }
  .err {
    color: var(--err);
    border-color: var(--err);
  }
  .warn {
    color: var(--warn);
    border-color: var(--warn);
  }
  .muted {
    color: var(--muted);
  }
</style>
