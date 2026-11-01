import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// The picker page is one script and one stylesheet inside internal/pick/assets,
// which the trama binary embeds. The files are committed, so nobody who only
// uses or builds trama needs Node. For development, run
// `trama add --pick <org> --no-open --port 7777` and then
// `npx vite -c vite.pick.config.ts`, and open /pick.html.
const target = `http://127.0.0.1:${process.env.TRAMA_PORT ?? "7777"}`;

export default defineConfig({
  plugins: [react()],
  define: { "process.env.NODE_ENV": JSON.stringify("production") },
  server: { proxy: { "/api": { target, changeOrigin: true } } },
  build: {
    outDir: "../internal/pick/assets",
    emptyOutDir: false,
    sourcemap: false,
    cssCodeSplit: false,
    lib: {
      entry: "src/pick/main.tsx",
      formats: ["iife"],
      name: "TramaPick",
      fileName: () => "pick.js",
      cssFileName: "pick",
    },
  },
});
