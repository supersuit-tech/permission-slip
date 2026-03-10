import { Routes, Route, Navigate, useSearchParams } from "react-router-dom";
import { SettingsNav } from "./SettingsNav";
import { ProfilePage } from "./ProfilePage";
import { SecurityPage } from "./SecurityPage";
import { IntegrationsPage } from "./IntegrationsPage";
import { BillingSettingsPage } from "./BillingSettingsPage";
import { AccountPage } from "./AccountPage";

/**
 * If the user lands on /settings with an oauth_status query param (from an
 * OAuth callback), redirect them to the integrations sub-page so the
 * ConnectedAccountsSection can pick it up.
 */
function SettingsIndex() {
  const [searchParams] = useSearchParams();
  if (searchParams.get("oauth_status")) {
    return <Navigate to={`/settings/integrations?${searchParams.toString()}`} replace />;
  }
  return <Navigate to="/settings/profile" replace />;
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
            <Route path="integrations" element={<IntegrationsPage />} />
            <Route path="billing" element={<BillingSettingsPage />} />
            <Route path="account" element={<AccountPage />} />
            <Route path="*" element={<Navigate to="/settings/profile" replace />} />
          </Routes>
        </div>
      </div>
    </div>
  );
}
