<script lang="ts">
  import Button from "./Button.svelte";

  let {
    label = "⭱ Import…",
    accept = ".yaml,.yml",
    onfile,
  }: {
    label?: string;
    accept?: string;
    onfile: (f: File) => void;
  } = $props();

  let el = $state<HTMLInputElement>() as HTMLInputElement;

  function onchange(e: Event) {
    const f = (e.currentTarget as HTMLInputElement).files?.[0];
    if (el) el.value = ""; // allow re-picking the same file
    if (f) onfile(f);
  }
</script>

<Button onclick={() => el?.click()}>{label}</Button>
<input bind:this={el} type="file" {accept} hidden {onchange} />
