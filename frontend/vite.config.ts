import path from "path";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
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
  },
});
