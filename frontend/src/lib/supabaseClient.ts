import { createClient } from "@supabase/supabase-js";

// Resolve the Supabase URL:
// 1. If VITE_SUPABASE_URL is set (cloud Supabase), use it directly.
// 2. Otherwise, use a relative /supabase path — the Go server proxies
//    /supabase/* → SUPABASE_URL. In dev mode, Vite's proxy handles the same
//    path. This eliminates CORS issues and avoids exposing extra ports,
//    which is especially useful for self-hosted and Raspberry Pi deployments.
const supabaseUrl =
  import.meta.env.VITE_SUPABASE_URL ||
  `${window.location.origin}/supabase`;

const supabasePublishableKey = import.meta.env.VITE_SUPABASE_PUBLISHABLE_KEY;

if (!supabasePublishableKey) {
  throw new Error(
    "Missing VITE_SUPABASE_PUBLISHABLE_KEY environment variable"
  );
}

export const supabase = createClient(supabaseUrl, supabasePublishableKey);
