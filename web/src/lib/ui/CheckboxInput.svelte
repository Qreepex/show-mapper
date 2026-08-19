<script lang="ts">
  let {
    checked = $bindable(false),
    label = "",
    onchange,
  }: {
    checked?: boolean;
    label?: string;
    onchange?: (e: Event) => void;
  } = $props();

  function handle(e: Event) {
    checked = (e.currentTarget as HTMLInputElement).checked;
    onchange?.(e);
  }
</script>

<label class="chk">
  <input type="checkbox" {checked} onchange={handle} />
  {#if label}<span>{label}</span>{/if}
</label>

<style>
  .chk {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.9rem;
    color: var(--text);
    cursor: pointer;
  }
</style>
