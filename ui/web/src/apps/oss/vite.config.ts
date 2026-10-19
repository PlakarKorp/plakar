import tailwindcss from "@tailwindcss/vite";
import { devtools } from "@tanstack/devtools-vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import react from "@vitejs/plugin-react";
import { createRequire } from "node:module";
import { visualizer } from "rollup-plugin-visualizer";
import { defineConfig } from "vite";

const require = createRequire(import.meta.url);

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
    tailwindcss(),
  ],
  resolve: {
    alias: {
      "~": "/src",
      path: require.resolve("path-browserify"),
    },
  },
});
