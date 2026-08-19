<script lang="ts">
  let {
    value = $bindable(""),
    placeholder = "",
    mono = false,
    grow = false,
    ...rest
  }: {
    value?: string;
    placeholder?: string;
    mono?: boolean;
    grow?: boolean;
    [key: string]: unknown;
  } = $props();
</script>

<input
  class="tin"
  class:mono
  class:grow
  type="text"
  bind:value
  {placeholder}
  {...rest}
/>

<style>
  .tin {
    background: #0c0f14;
    color: var(--text);
    border: 1px solid var(--border);
    border-radius: 0.35rem;
    padding: 0.35rem 0.5rem;
    font: inherit;
    width: 100%;
  }
  .tin:focus {
    outline: 1px solid var(--accent);
  }
  .mono {
    font-family: ui-monospace, "Cascadia Code", monospace;
    font-size: 0.85em;
  }
</style>
