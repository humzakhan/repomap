import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  build: {
    outDir: "dist",
    emptyOutDir: true,
    rollupOptions: {
      input: "index.html",
      output: {
        format: "iife",
        entryFileNames: "bundle.js",
        assetFileNames: (assetInfo) => {
          if (assetInfo.names?.[0]?.endsWith(".css")) {
            return "styles.css";
          }
          return "[name].[ext]";
        },
        inlineDynamicImports: true,
      },
    },
    cssCodeSplit: false,
  },
  server: {
    proxy: {
      "/api": "http://localhost:4242",
    },
  },
});
