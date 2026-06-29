import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import tailwindcss from "@tailwindcss/vite";

// The Rust binary embeds frontend/dist and serves it from `/`, so asset URLs
// must be relative (base "./") rather than rooted at "/".
export default defineConfig({
  base: "./",
  plugins: [svelte(), tailwindcss()],
  server: {
    proxy: {
      // The live Go backend (identical API) runs here during development.
      "/api": {
        target: "http://127.0.0.1:7799",
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
});
