/// <reference types="vite/client" />

/** Vite environment variables. See .env.example for defaults. */
interface ImportMetaEnv {
  /** Base URL for API calls (default: "/api"). */
  readonly VITE_API_BASE_URL?: string;
  /** Supabase project URL (required). */
  readonly VITE_SUPABASE_URL: string;
  /** Supabase publishable key (required). */
  readonly VITE_SUPABASE_PUBLISHABLE_KEY: string;
  /** Set to "true" to load the standalone React DevTools connector (dev only). */
  readonly VITE_REACT_DEVTOOLS?: string;
  /** Sentry DSN for error tracking. Omit to disable Sentry. */
  readonly VITE_SENTRY_DSN?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
