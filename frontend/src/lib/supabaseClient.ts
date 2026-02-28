import { createClient } from "@supabase/supabase-js";

// In development, resolve Supabase via a relative path so requests always go
// to the same origin as the page — whether that's localhost:5173, the dev
// proxy on :3000, or the ngrok tunnel. The dev proxy routes /supabase/* →
// http://127.0.0.1:54321. This avoids cross-origin CORS failures when the
// browser is on a different host than VITE_SUPABASE_URL points to.
//
// In production, VITE_SUPABASE_URL must be set to the real Supabase project URL.
const supabaseUrl =
  import.meta.env.DEV
    ? `${window.location.origin}/supabase`
    : import.meta.env.VITE_SUPABASE_URL;

const supabaseAnonKey = import.meta.env.VITE_SUPABASE_ANON_KEY;

if (!supabaseUrl || !supabaseAnonKey) {
  throw new Error(
    "Missing VITE_SUPABASE_URL or VITE_SUPABASE_ANON_KEY environment variables"
  );
}

export const supabase = createClient(supabaseUrl, supabaseAnonKey);
