import react from "@vitejs/plugin-react";
import { defineConfig, type Plugin } from "vite";

// With TRAMA_MOCK=1 the dev server answers /api/workspace with a made-up
// workspace (TRAMA_MOCK_REPOS=100 for a big one), so the page can be worked on without a real one. Otherwise the
// request goes to `trama graph --serve --no-open --port 7777`.
function mock(): Plugin {
  return {
    name: "trama-mock",
    configureServer(server) {
      if (!process.env.TRAMA_MOCK) return;
      server.middlewares.use("/api/workspace", async (_req, res) => {
        const { sampleModel, bigModel } = await server.ssrLoadModule("/dev/sample.ts");
        const n = Number(process.env.TRAMA_MOCK_REPOS);
        res.setHeader("Content-Type", "application/json");
        res.end(JSON.stringify(n > 0 ? bigModel(n) : sampleModel()));
      });
    },
  };
}

const target = `http://127.0.0.1:${process.env.TRAMA_PORT ?? "7777"}`;

export default defineConfig({
  plugins: [react(), mock()],
  define: { "process.env.NODE_ENV": JSON.stringify("production") },
  server: { proxy: { "/api": { target, changeOrigin: true } } },
  // The build is one script and one stylesheet inside internal/view/assets,
  // which the trama binary embeds. The files are committed, so nobody who
  // only uses or builds trama needs Node.
  build: {
    outDir: "../internal/view/assets",
    emptyOutDir: false,
    sourcemap: false,
    cssCodeSplit: false,
    lib: {
      entry: "src/main.tsx",
      formats: ["iife"],
      name: "TramaGraph",
      fileName: () => "app.js",
      cssFileName: "app",
    },
  },
});
