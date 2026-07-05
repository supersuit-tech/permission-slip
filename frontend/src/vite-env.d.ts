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
  /** Git commit SHA stamped at build time (full hash). */
  readonly VITE_GIT_COMMIT_HASH?: string;
  /** Git commit timestamp stamped at build time (ISO-8601). */
  readonly VITE_GIT_COMMIT_TIMESTAMP?: string;
  /** Web release version from the latest web/v* tag (stamped at build time). */
  readonly VITE_WEB_VERSION?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
