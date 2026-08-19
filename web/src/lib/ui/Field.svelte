<script lang="ts">
  import type { Snippet } from "svelte";
  let {
    label,
    hint,
    grow = false,
    children,
  }: {
    label: string;
    hint?: string;
    grow?: boolean;
    children?: Snippet;
  } = $props();
</script>

<div class="field" class:grow>
  <span class="flabel">{label}</span>
  {@render children?.()}
  {#if hint}<span class="fhint">{hint}</span>{/if}
</div>

<style>
  .field {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    min-width: 8rem;
  }
  .field.grow {
    flex: 1;
    min-width: 12rem;
  }
  .flabel {
    font-size: 0.78rem;
    color: var(--muted);
  }
  .fhint {
    font-size: 0.75rem;
    color: var(--muted);
    opacity: 0.8;
  }
</style>
