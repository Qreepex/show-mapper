<script lang="ts">
  import { page } from "$app/state";
  import "../app.css";
  import { live } from "$lib/ws.svelte";

  let { children } = $props();

  $effect(() => {
    // module effect runs once in the browser (SPA mode)
    live.connect();
  });

  const routes = [
    { href: "/", label: "Dashboard" },
    { href: "/mappings", label: "Mappings" },
    { href: "/settings", label: "Settings" },
  ];

  const wsBadge = $derived(
    live.ws === "open" ? "ok" : live.ws === "closed" ? "err" : "warn",
  );
</script>

<nav class="top">
  <strong>showbridge</strong>
  {#each routes as r (r.href)}
    <a href={r.href} class:active={page.url.pathname === r.href}>{r.label}</a>
  {/each}
  <span class="grow"></span>
  <span class="badge {wsBadge}">ws: {live.ws}</span>
  <span class="muted mono">v{live.version || "?"}</span>
</nav>

<main class="page">
  {@render children()}
</main>

<style>
  .top {
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 0.6rem 1rem;
    background: #0b0d11;
    border-bottom: 1px solid var(--border);
  }
  .top strong { color: var(--accent); }
  .top a {
    color: var(--muted);
    text-decoration: none;
    padding: 0.2rem 0.55rem;
    border-radius: 0.35rem;
  }
  .top a.active { color: var(--text); background: var(--panel-2); }
  .grow { flex: 1; }
  .muted { color: var(--muted); }
  .mono { font-family: ui-monospace, monospace; font-size: 0.8rem; }
</style>
