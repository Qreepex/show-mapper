<script lang="ts">
  import type { Snippet } from "svelte";
  let {
    variant = "default",
    children,
    ...rest
  }: {
    variant?: "default" | "primary" | "danger" | "ghost";
    children?: Snippet;
    [key: string]: unknown;
  } = $props();
</script>

<button class="btn btn-{variant}" {...rest}>{@render children?.()}</button>

<style>
  .btn {
    background: var(--panel-2);
    color: var(--text);
    border: 1px solid var(--border);
    border-radius: 0.4rem;
    padding: 0.4rem 0.8rem;
    cursor: pointer;
    font: inherit;
    white-space: nowrap;
  }
  .btn:hover:not(:disabled) {
    border-color: var(--accent);
  }
  .btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .btn-primary {
    background: #1d3b53;
    border-color: #2c5878;
  }
  .btn-danger {
    color: var(--err);
  }
  .btn-ghost {
    background: transparent;
    border-color: transparent;
  }
  .btn-ghost:hover:not(:disabled) {
    border-color: var(--border);
  }
</style>
