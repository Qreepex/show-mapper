<script lang="ts">
  let { msg }: { msg?: { text: string; ok: boolean } | null } = $props();
</script>

{#if msg}
  <span class={msg.ok ? "msg ok" : "msg err"}>{msg.text}</span>
{/if}

<style>
  .msg {
    font-size: 0.85rem;
    white-space: pre-wrap;
  }
  .ok {
    color: var(--ok);
  }
  .err {
    color: var(--err);
  }
</style>
