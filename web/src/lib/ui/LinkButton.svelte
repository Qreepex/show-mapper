<script lang="ts">
  import type { Snippet } from "svelte";
  let {
    href,
    download,
    variant = "default",
    children,
  }: {
    href: string;
    download?: string | boolean;
    variant?: "default" | "primary";
    children?: Snippet;
  } = $props();
</script>

<a class="lbtn lbtn-{variant}" {href} download={download === true ? "" : download}>
  {@render children?.()}
</a>

<style>
  .lbtn {
    display: inline-block;
    background: var(--panel-2);
    color: var(--text);
    border: 1px solid var(--border);
    border-radius: 0.4rem;
    padding: 0.4rem 0.8rem;
    text-decoration: none;
    white-space: nowrap;
  }
  .lbtn:hover {
    border-color: var(--accent);
  }
  .lbtn-primary {
    background: #1d3b53;
    border-color: #2c5878;
  }
</style>
