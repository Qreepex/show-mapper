<script lang="ts">
  import TopNav from "$lib/ui/TopNav.svelte";
  import "../app.css";
  import { live } from "$lib/ws.svelte";

  let { children } = $props();

  $effect(() => {
    // module effect runs once in the browser (SPA mode)
    live.connect();
  });
</script>

<TopNav />
<main class="wrap">
  {@render children()}
</main>
