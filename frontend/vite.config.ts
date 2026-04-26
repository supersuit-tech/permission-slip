import path from "path";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { sentryVitePlugin } from "@sentry/vite-plugin";

// Source maps are only generated when all Sentry upload vars are present.
// This prevents .map files from being left in dist/ and accidentally served
// publicly by the Go SPA handler when the upload step is skipped.
const sentryUploadEnabled = Boolean(
  process.env.SENTRY_AUTH_TOKEN &&
    process.env.SENTRY_ORG &&
    process.env.SENTRY_PROJECT,
);

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    // Sentry source-map upload runs only during production builds when the
    // required env vars are set. Place after other plugins so Sentry can
    // process the final bundle output.
    sentryVitePlugin({
      org: process.env.SENTRY_ORG,
      project: process.env.SENTRY_PROJECT,
      authToken: process.env.SENTRY_AUTH_TOKEN,
      sourcemaps: {
        filesToDeleteAfterUpload: ["./dist/**/*.map"],
      },
      disable: !sentryUploadEnabled,
    }),
  ],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
      "@config": path.resolve(__dirname, "../config"),
    },
  },
  server: {
    fs: {
      allow: [
        // Frontend project root (Vite's default — must be explicit when fs.allow is set).
        path.resolve(__dirname),
        // Allow Vite dev server to read config/ from repo root.
        path.resolve(__dirname, "../config"),
      ],
    },
    // Proxy routes for local dev (npm run dev). Keep in sync with
    // dev-proxy.cjs which mirrors these routes for ngrok tunneling.
    proxy: {
      "/api": "http://localhost:8080",
      "/invite": "http://localhost:8080",
      "/supabase": {
        target: "http://127.0.0.1:54321",
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/supabase/, ""),
      },
      "/mailpit": {
        target: "http://127.0.0.1:54324",
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/mailpit/, ""),
      },
    },
    allowedHosts: [".ngrok-free.dev"],
  },
  build: {
    outDir: "dist",
    // Generate source maps only when Sentry upload is configured.
    // "hidden" generates .map files without adding sourceMappingURL comments
    // to the bundles. The Sentry plugin uploads then deletes them.
    // When upload is not configured, skip generation entirely so .map files
    // don't end up in dist/ and get served publicly.
    sourcemap: sentryUploadEnabled ? "hidden" : false,
  },
});
