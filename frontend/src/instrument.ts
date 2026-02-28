import * as Sentry from "@sentry/react";
import React from "react";
import {
  useLocation,
  useNavigationType,
  createRoutesFromChildren,
  matchRoutes,
} from "react-router-dom";

const dsn = import.meta.env.VITE_SENTRY_DSN;

if (dsn) {
  // Escape regex special chars in the origin (e.g. dots in hostnames)
  // so the pattern matches the literal origin, not wildcards.
  const escapedOrigin = window.location.origin.replace(
    /[.*+?^${}()|[\]\\]/g,
    "\\$&",
  );

  Sentry.init({
    dsn,
    environment: import.meta.env.MODE, // "development" | "production"
    // Release is automatically injected by @sentry/vite-plugin at build
    // time — no need to set it here.
    integrations: [
      Sentry.reactRouterV7BrowserTracingIntegration({
        useEffect: React.useEffect,
        useLocation,
        useNavigationType,
        createRoutesFromChildren,
        matchRoutes,
      }),
    ],
    // 10% of transactions for performance monitoring (free-tier friendly)
    tracesSampleRate: 0.1,
    // Attach trace headers to same-origin API requests so backend spans
    // are correlated. Match both absolute (openapi-fetch resolves to
    // origin/api) and relative (/api) URLs, but avoid matching Sentry's
    // own ingest endpoint (*.ingest.sentry.io/api/...).
    tracePropagationTargets: [
      new RegExp(`^${escapedOrigin}/api`),
      /^\/api/,
    ],
  });
}
