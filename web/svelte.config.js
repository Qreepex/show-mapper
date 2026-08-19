import adapter from "@sveltejs/adapter-static";
import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";

/** @type {import('@sveltejs/kit').Config} */
const config = {
  preprocess: vitePreprocess(),
  kit: {
    // SPA, built straight into the Go package dir so `go build` embeds it
    // (internal/server/embed.go). Run `make web` (or npm run build) before go build.
    adapter: adapter({
      pages: "../internal/server/dist",
      assets: "../internal/server/dist",
      fallback: "200.html",
    }),
  },
};

export default config;
