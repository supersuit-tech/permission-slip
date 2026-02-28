// Sentry must be initialized before any other imports so it can instrument
// global handlers (fetch, console, etc.) before they are used.
import "./instrument";

import React from "react";
import ReactDOM from "react-dom/client";
import * as Sentry from "@sentry/react";

// Opt-in standalone React DevTools connection (useful for remote/ngrok sessions).
// Enable by setting VITE_REACT_DEVTOOLS=true in frontend/.env, then run:
//   npm run devtools   (opens a dedicated DevTools window on port 8097)
// Without this flag, the browser extension works automatically in dev mode.
if (import.meta.env.DEV && import.meta.env.VITE_REACT_DEVTOOLS === "true") {
  const script = document.createElement("script");
  script.src = "http://localhost:8097";
  script.async = true;
  document.head.appendChild(script);
}
import {
  MutationCache,
  QueryCache,
  QueryClient,
  QueryClientProvider,
} from "@tanstack/react-query";
import { BrowserRouter } from "react-router-dom";
import { AuthProvider } from "./auth/AuthContext";
import { CookieConsentBanner } from "./components/CookieConsentBanner";
import { CookieConsentProvider } from "./components/CookieConsentContext";
import { ThemeProvider } from "./components/ThemeContext";
import { Toaster } from "./components/ui/sonner";
import App from "./App";
import "./index.css";

function safeStringify(value: unknown): string {
  try {
    return JSON.stringify(value);
  } catch {
    return "[unserializable]";
  }
}

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // Default staleTime of 0 causes refetches on every window focus,
      // mount, and reconnect — even when the data hasn't changed. 30s is
      // a reasonable baseline that avoids unnecessary network requests
      // during rapid state transitions (e.g. MFA enrollment) while still
      // keeping data reasonably fresh. Individual queries can override.
      staleTime: 30_000,
    },
  },
  queryCache: new QueryCache({
    onError(error, query) {
      Sentry.addBreadcrumb({
        category: "react-query.query",
        message: error instanceof Error ? error.message : String(error),
        level: "error",
        data: { queryKey: safeStringify(query.queryKey) },
      });
    },
  }),
  mutationCache: new MutationCache({
    onError(error) {
      Sentry.addBreadcrumb({
        category: "react-query.mutation",
        message: error instanceof Error ? error.message : String(error),
        level: "error",
      });
    },
  }),
});

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <BrowserRouter>
      <QueryClientProvider client={queryClient}>
        <ThemeProvider>
          <CookieConsentProvider>
            <AuthProvider>
              <App />
              <CookieConsentBanner />
              <Toaster />
            </AuthProvider>
          </CookieConsentProvider>
        </ThemeProvider>
      </QueryClientProvider>
    </BrowserRouter>
  </React.StrictMode>
);
