import { Loader2 } from "lucide-react";
import { Routes, Route, Navigate, useLocation } from "react-router-dom";
import { useAuth } from "./auth/AuthContext";
import LoginPage from "./auth/LoginPage";
import MfaChallengePage from "./auth/MfaChallengePage";
import OnboardingPage from "./auth/OnboardingPage";
import { AppLayout } from "./components/AppLayout";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { Dashboard } from "./pages/dashboard/Dashboard";
import { AgentConfigPage } from "./pages/agents/AgentConfigPage";
import { ConnectorConfigPage } from "./pages/agents/connectors/ConnectorConfigPage";
import { ActivityPage } from "./pages/activity/ActivityPage";
import { SettingsPage } from "./pages/settings/SettingsPage";
import { PrivacyPolicyPage } from "./pages/policy/PrivacyPolicyPage";
import { TermsOfServicePage } from "./pages/policy/TermsOfServicePage";
import { CookiePolicyPage } from "./pages/policy/CookiePolicyPage";
import { useProfile } from "./hooks/useProfile";

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
  const { authStatus } = useAuth();
  const { needsOnboarding, isLoading: profileLoading } = useProfile();

  // Policy pages are public — render without auth.
  if (pathname.startsWith("/policy/")) {
    return (
      <Routes>
        <Route path="/policy/privacy" element={<PrivacyPolicyPage />} />
        <Route path="/policy/terms" element={<TermsOfServicePage />} />
        <Route path="/policy/cookies" element={<CookiePolicyPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
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
    <ErrorBoundary>
      <AppLayout>
        <Routes>
          <Route path="/" element={<ErrorBoundary fallback={<RouteErrorFallback />}><Dashboard /></ErrorBoundary>} />
          <Route path="/agents/:agentId" element={<ErrorBoundary fallback={<RouteErrorFallback />}><AgentConfigPage /></ErrorBoundary>} />
          <Route path="/agents/:agentId/connectors/:connectorId" element={<ErrorBoundary fallback={<RouteErrorFallback />}><ConnectorConfigPage /></ErrorBoundary>} />
          <Route path="/activity" element={<ErrorBoundary fallback={<RouteErrorFallback />}><ActivityPage /></ErrorBoundary>} />
          <Route path="/settings" element={<ErrorBoundary fallback={<RouteErrorFallback />}><SettingsPage /></ErrorBoundary>} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </AppLayout>
    </ErrorBoundary>
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
