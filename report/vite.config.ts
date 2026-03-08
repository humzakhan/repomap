import { defineConfig } from "vite";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [tailwindcss()],
  build: {
    outDir: "dist",
    emptyOutDir: true,
    rollupOptions: {
      input: "src/main.ts",
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
});
