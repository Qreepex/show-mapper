<script lang="ts">
  export type SelectOption = string | { value: string; label?: string };

  let {
    value = $bindable(""),
    options = [],
    allowEmpty = false,
    onchange,
  }: {
    value?: string;
    options?: SelectOption[];
    allowEmpty?: boolean | string;
    onchange?: (e: Event) => void;
  } = $props();

  const opts = $derived(
    options.map((o) => (typeof o === "string" ? { value: o, label: o } : o)),
  );

  function handle(e: Event) {
    value = (e.currentTarget as HTMLSelectElement).value;
    onchange?.(e);
  }
</script>

<select class="sel" value={value} onchange={handle}>
  {#if allowEmpty}
    <option value="">{allowEmpty === true ? "(auto)" : allowEmpty}</option>
  {/if}
  {#each opts as o (o.value)}
    <option value={o.value}>{o.label}</option>
  {/each}
</select>

<style>
  .sel {
    background: #0c0f14;
    color: var(--text);
    border: 1px solid var(--border);
    border-radius: 0.35rem;
    padding: 0.35rem 0.5rem;
    font: inherit;
    max-width: 22rem;
  }
  .sel:focus {
    outline: 1px solid var(--accent);
  }
</style>
