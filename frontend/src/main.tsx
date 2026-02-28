import React from "react";
import ReactDOM from "react-dom/client";

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
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter } from "react-router-dom";
import { AuthProvider } from "./auth/AuthContext";
import { CookieConsentBanner } from "./components/CookieConsentBanner";
import { CookieConsentProvider } from "./components/CookieConsentContext";
import { ThemeProvider } from "./components/ThemeContext";
import { Toaster } from "./components/ui/sonner";
import App from "./App";
import "./index.css";

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
