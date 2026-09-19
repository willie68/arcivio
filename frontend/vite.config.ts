import { defineConfig, type ProxyOptions } from "vite";
import vue from "@vitejs/plugin-vue";
import { resolve } from "node:path";

const backend = process.env.VITE_BACKEND_URL ?? "https://127.0.0.1:9443";

const backendProxy: ProxyOptions = {
  target: backend,
  secure: false,
  changeOrigin: true,
  configure(proxy) {
    proxy.on("proxyReq", (proxyReq, req) => {
      if (!req.headers["x-forwarded-proto"]) {
        proxyReq.setHeader("X-Forwarded-Proto", "http");
      }
    });
  },
};

export default defineConfig(({ command }) => ({
  plugins: [vue()],
  // Production HTML is served from `/`, assets from `/client/`.
  // Vite-dev must use `/`, otherwise `/src/main.ts` 404s and the page stays blank.
  base: command === "build" ? "/client/" : "/",
  server: {
    port: 5173,
    proxy: {
      "/auth": backendProxy,
      "/api": backendProxy,
      "/swagger": backendProxy,
      "/livez": backendProxy,
      "/readyz": backendProxy,
    },
  },
  build: {
    outDir: resolve(__dirname, "../backend/pkg/web/client"),
    emptyOutDir: true,
  },
}));
