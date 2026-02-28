import * as Sentry from "@sentry/react";
import { type ReactNode, useEffect } from "react";
import { Loader2 } from "lucide-react";
import { Routes, Route, Navigate, useLocation } from "react-router-dom";
import { useAuth } from "./auth/AuthContext";
import LoginPage from "./auth/LoginPage";
import MfaChallengePage from "./auth/MfaChallengePage";
import OnboardingPage from "./auth/OnboardingPage";
import { AppLayout } from "./components/AppLayout";
import { Dashboard } from "./pages/dashboard/Dashboard";
import { AgentConfigPage } from "./pages/agents/AgentConfigPage";
import { ConnectorConfigPage } from "./pages/agents/connectors/ConnectorConfigPage";
import { ActivityPage } from "./pages/activity/ActivityPage";
import { SettingsPage } from "./pages/settings/SettingsPage";
import { PrivacyPolicyPage } from "./pages/policy/PrivacyPolicyPage";
import { TermsOfServicePage } from "./pages/policy/TermsOfServicePage";
import { CookiePolicyPage } from "./pages/policy/CookiePolicyPage";
import { useProfile } from "./hooks/useProfile";

const SentryRoutes = Sentry.withSentryReactRouterV7Routing(Routes);

/** Full-page placeholder shown while auth or profile data is loading. */
function LoadingFallback() {
  return (
    <div role="status" aria-label="Loading" className="flex min-h-screen flex-col items-center justify-center gap-4">
      <span className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary text-lg font-bold text-primary-foreground">
        P
      </span>
      <p className="font-serif text-xl font-semibold tracking-tight">
        Permission Slip
      </p>
      <Loader2 className="text-muted-foreground size-5 animate-spin" />
    </div>
  );
}

/**
 * Root routing component. Renders a single page based on the user's
 * authentication and onboarding state, evaluated in this order:
 *
 *  0. /policy/*        → Public policy pages (no auth required)
 *  1. loading          → LoadingFallback (auth session resolving)
 *  2. mfa_required     → MfaChallengePage (TOTP challenge before dashboard)
 *  3. unauthenticated  → LoginPage
 *  4. profile loading  → LoadingFallback (fetching profile after auth)
 *  5. needs onboarding → OnboardingPage (first-time user, no profile yet)
 *  6. authenticated    → Routes inside AppLayout
 */
function App() {
  const { pathname } = useLocation();
  const { authStatus, user } = useAuth();
  const { needsOnboarding, isLoading: profileLoading } = useProfile();

  // Set Sentry user context so errors include the user's identity.
  // Only the opaque Supabase user ID is sent — no email or PII.
  useEffect(() => {
    if (authStatus === "authenticated" && user) {
      Sentry.setUser({ id: user.id });
    } else {
      Sentry.setUser(null);
    }
  }, [authStatus, user]);

  // Policy pages are public — render without auth.
  if (pathname.startsWith("/policy/")) {
    return (
      <SentryRoutes>
        <Route path="/policy/privacy" element={<PrivacyPolicyPage />} />
        <Route path="/policy/terms" element={<TermsOfServicePage />} />
        <Route path="/policy/cookies" element={<CookiePolicyPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </SentryRoutes>
    );
  }

  if (authStatus === "loading") {
    return <LoadingFallback />;
  }

  if (authStatus === "mfa_required") {
    return <MfaChallengePage />;
  }

  if (authStatus !== "authenticated") {
    return <LoginPage />;
  }

  if (profileLoading) {
    return <LoadingFallback />;
  }

  if (needsOnboarding) {
    return <OnboardingPage />;
  }

  return (
    <Sentry.ErrorBoundary fallback={<AppCrashFallback />} showDialog>
      <AppLayout>
        <SentryRoutes>
          <Route path="/" element={<RouteWithBoundary><Dashboard /></RouteWithBoundary>} />
          <Route path="/agents/:agentId" element={<RouteWithBoundary><AgentConfigPage /></RouteWithBoundary>} />
          <Route path="/agents/:agentId/connectors/:connectorId" element={<RouteWithBoundary><ConnectorConfigPage /></RouteWithBoundary>} />
          <Route path="/activity" element={<RouteWithBoundary><ActivityPage /></RouteWithBoundary>} />
          <Route path="/settings" element={<RouteWithBoundary><SettingsPage /></RouteWithBoundary>} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </SentryRoutes>
      </AppLayout>
    </Sentry.ErrorBoundary>
  );
}

/** Wraps a route's content in Sentry.ErrorBoundary so a crash on one page
 *  doesn't take down the entire app — the nav and layout remain usable. */
function RouteWithBoundary({ children }: { children: ReactNode }) {
  return (
    <Sentry.ErrorBoundary fallback={<RouteErrorFallback />}>
      {children}
    </Sentry.ErrorBoundary>
  );
}

/** Full-page crash fallback when the outer ErrorBoundary catches an error. */
function AppCrashFallback() {
  return (
    <div className="flex min-h-screen items-center justify-center p-8">
      <div className="text-center">
        <h1 className="mb-2 text-2xl font-semibold">Something went wrong</h1>
        <p className="text-muted-foreground mb-4">
          An unexpected error occurred. Please try refreshing the page.
        </p>
        <button
          onClick={() => window.location.reload()}
          className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
        >
          Refresh Page
        </button>
      </div>
    </div>
  );
}

/** Inline error fallback that preserves the AppLayout (header, nav). */
function RouteErrorFallback() {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <h2 className="mb-2 text-lg font-semibold">Something went wrong</h2>
      <p className="text-muted-foreground mb-4 text-sm">
        An unexpected error occurred on this page.
      </p>
      <button
        onClick={() => window.location.reload()}
        className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
      >
        Refresh Page
      </button>
    </div>
  );
}

export default App;
