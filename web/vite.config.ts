import { sveltekit } from "@sveltejs/kit/vite";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [sveltekit()],
  server: {
    port: 5173,
    proxy: {
      // dev: go backend runs via `go run ./cmd/show-mapper` (or a release binary)
      "/api": "http://localhost:8484",
      "/ws": { target: "ws://localhost:8484", ws: true },
    },
  },
});
