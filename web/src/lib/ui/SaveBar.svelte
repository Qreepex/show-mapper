<script lang="ts">
  import type { Snippet } from "svelte";
  import Button from "./Button.svelte";
  import Msg from "./Msg.svelte";

  let {
    onsave,
    msg,
    children,
  }: {
    onsave: () => void;
    msg?: { text: string; ok: boolean } | null;
    children?: Snippet;
  } = $props();
</script>

<div class="savebar">
  <Button variant="primary" onclick={onsave}>Save &amp; apply</Button>
  {@render children?.()}
  <Msg {msg} />
</div>

<style>
  .savebar {
    display: flex;
    gap: 0.6rem;
    align-items: center;
    flex-wrap: wrap;
    margin-bottom: 1rem;
  }
</style>
