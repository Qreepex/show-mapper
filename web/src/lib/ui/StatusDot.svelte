<script lang="ts">
  let { state = "disconnected" }: { state?: string } = $props();
</script>

<span class={["dot", state]} title={state}></span>

<style>
  .dot {
    display: inline-block;
    width: 0.6rem;
    height: 0.6rem;
    border-radius: 50%;
    flex: none;
  }
  .connected {
    background: var(--ok);
  }
  .error {
    background: var(--err);
  }
  .connecting {
    background: var(--warn);
  }
  .disconnected {
    background: var(--muted);
  }
</style>
