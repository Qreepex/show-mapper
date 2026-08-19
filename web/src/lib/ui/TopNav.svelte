<script lang="ts">
  import { page } from "$app/state";
  import { live } from "$lib/ws.svelte";
  import Badge from "./Badge.svelte";

  const routes = [
    { href: "/", label: "Dashboard" },
    { href: "/surface", label: "Surface" },
    { href: "/mappings", label: "Mappings" },
    { href: "/sources", label: "Sources" },
    { href: "/targets", label: "Targets" },
    { href: "/boards", label: "Boards" },
    { href: "/settings", label: "Settings" },
  ];

  const wsVariant = $derived(
    live.ws === "open" ? "ok" : live.ws === "closed" ? "err" : "warn",
  );
</script>

<nav class="top">
  <a class="brand" href="/">show-mapper</a>
  {#each routes as r (r.href)}
    <a class="link" class:active={page.url.pathname === r.href} href={r.href}>{r.label}</a>
  {/each}
  <span class="spacer"></span>
  <Badge variant={wsVariant}>ws: {live.ws}</Badge>
  <span class="ver">v{live.version || "?"}</span>
</nav>

<style>
  .top {
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 0.6rem 1rem;
    background: #0b0d11;
    border-bottom: 1px solid var(--border);
    position: sticky;
    top: 0;
    z-index: 10;
  }
  .brand {
    color: var(--accent);
    font-weight: 700;
    text-decoration: none;
  }
  .link {
    color: var(--muted);
    text-decoration: none;
    padding: 0.2rem 0.55rem;
    border-radius: 0.35rem;
    font-size: 0.95rem;
  }
  .link.active {
    color: var(--text);
    background: var(--panel-2);
  }
  .spacer {
    flex: 1;
  }
  .ver {
    color: var(--muted);
    font-family: ui-monospace, monospace;
    font-size: 0.8rem;
  }
</style>
