<script lang="ts">
  let {
    value = $bindable(),
    min,
    max,
    step = "any",
    ...rest
  }: {
    value?: number;
    min?: number;
    max?: number;
    step?: number | "any";
    [key: string]: unknown;
  } = $props();
</script>

<input class="nin" type="number" bind:value {min} {max} {step} {...rest} />

<style>
  .nin {
    background: #0c0f14;
    color: var(--text);
    border: 1px solid var(--border);
    border-radius: 0.35rem;
    padding: 0.35rem 0.5rem;
    font: inherit;
    width: 7rem;
  }
  .nin:focus {
    outline: 1px solid var(--accent);
  }
</style>
