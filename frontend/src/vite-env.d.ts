/// <reference types="vite/client" />

/** Vite environment variables. See .env.example for defaults. */
interface ImportMetaEnv {
  /** Base URL for API calls (default: "/api"). */
  readonly VITE_API_BASE_URL?: string;
  /** Supabase project URL (required). */
  readonly VITE_SUPABASE_URL: string;
  /** Supabase anonymous/public key (required). */
  readonly VITE_SUPABASE_ANON_KEY: string;
  /** Set to "true" to load the standalone React DevTools connector (dev only). */
  readonly VITE_REACT_DEVTOOLS?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
