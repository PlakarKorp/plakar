import tailwindcss from "@tailwindcss/vite";
import { devtools } from "@tanstack/devtools-vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import react from "@vitejs/plugin-react";
import { visualizer } from "rollup-plugin-visualizer";
import { defineConfig } from "vite";
import { nodePolyfills } from "vite-plugin-node-polyfills";

// https://vite.dev/config/
export default defineConfig({
  build: {
    chunkSizeWarningLimit: 1024 * 1024, // 1MB
  },
  plugins: [
    ...(process.env.VITE_DEVTOOLS ? [devtools()] : []),
    tanstackRouter({
      target: "react",
      autoCodeSplitting: true,
    }),
    react(),
    visualizer({
      filename: "stats.html",
      template: "flamegraph",
      gzipSize: true,
    }), // On build, create stats.html to visualize the bundle size
    nodePolyfills({
      include: ["path"],
    }),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      "~": "/src",
    },
  },
});
