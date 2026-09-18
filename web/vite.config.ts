import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import { resolve } from "node:path";

export default defineConfig({
  plugins: [vue()],
  base: "/client/",
  build: {
    outDir: resolve(__dirname, "../pkg/web/client"),
    emptyOutDir: true,
  },
});
