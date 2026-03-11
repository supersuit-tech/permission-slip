import { useEffect, useRef } from "react";
import { Routes, Route, Navigate, useSearchParams, useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { providerLabel } from "@/lib/oauth";
import { SettingsNav } from "./SettingsNav";
import { ProfilePage } from "./ProfilePage";
import { SecurityPage } from "./SecurityPage";
import { BillingSettingsPage } from "./BillingSettingsPage";
import { AccountPage } from "./AccountPage";
import { IntegrationsPage } from "./IntegrationsPage";

/**
 * Handles the OAuth callback redirect from the backend. The backend sends
 * users to /settings?oauth_status=success&oauth_provider=github after OAuth
 * completes. This component shows a toast and redirects to the appropriate
 * settings sub-page (integrations for OAuth callbacks, profile otherwise).
 */
function SettingsIndex() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const firedRef = useRef(false);

  useEffect(() => {
    if (firedRef.current) return;
    firedRef.current = true;

    const oauthStatus = searchParams.get("oauth_status");
    const oauthProvider = searchParams.get("oauth_provider");
    if (oauthStatus) {
      if (oauthStatus === "success") {
        toast.success(
          `Successfully connected ${oauthProvider ? providerLabel(oauthProvider) : "account"}.`,
        );
      } else {
        const oauthError = searchParams.get("oauth_error");
        const label = oauthProvider
          ? providerLabel(oauthProvider)
          : "account";
        const detail = oauthError
          ? `Failed to connect ${label}: ${oauthError}`
          : `Failed to connect ${label}. Please try again.`;
        toast.error(detail);
      }
    }
    const oauthTab = searchParams.get("oauth_tab");
    const dest = oauthTab === "connections" ? "/settings/integrations" : "/settings/profile";
    navigate(dest, { replace: true });
  }, []); // eslint-disable-line react-hooks/exhaustive-deps -- run once on mount

  return null;
}

export function SettingsLayout() {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold tracking-tight">Settings</h1>

      <div className="flex flex-col gap-6 md:flex-row">
        <SettingsNav />
        <div className="flex-1 min-w-0 space-y-6">
          <Routes>
            <Route index element={<SettingsIndex />} />
            <Route path="profile" element={<ProfilePage />} />
            <Route path="security" element={<SecurityPage />} />
            <Route path="billing" element={<BillingSettingsPage />} />
            <Route path="account" element={<AccountPage />} />
            <Route path="integrations" element={<IntegrationsPage />} />
            <Route path="*" element={<Navigate to="/settings/profile" replace />} />
          </Routes>
        </div>
      </div>
    </div>
  );
}
