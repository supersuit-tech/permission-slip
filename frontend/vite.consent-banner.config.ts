import { defineConfig } from "vite";
import path from "path";

/**
 * Separate Vite build config for the embeddable consent banner script.
 *
 * Produces a single self-contained JS file (no external dependencies) that
 * can be loaded via <script src="consent-banner.js"></script> on any page.
 *
 * Build: npx vite build --config vite.consent-banner.config.ts
 */
export default defineConfig({
  build: {
    lib: {
      entry: path.resolve(__dirname, "src/consent-banner/embed.ts"),
      name: "PSConsentBanner",
      fileName: () => "consent-banner.js",
      formats: ["iife"],
    },
    outDir: "dist",
    emptyOutDir: false,
    rollupOptions: {
      output: {
        // Inline everything — no external chunks.
        inlineDynamicImports: true,
      },
    },
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
});
