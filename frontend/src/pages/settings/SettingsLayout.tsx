import { Routes, Route, Navigate } from "react-router-dom";
import { SettingsNav } from "./SettingsNav";
import { ProfilePage } from "./ProfilePage";
import { SecurityPage } from "./SecurityPage";
import { BillingSettingsPage } from "./BillingSettingsPage";
import { AccountPage } from "./AccountPage";
import { IntegrationsPage } from "./IntegrationsPage";

export function SettingsLayout() {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold tracking-tight">Settings</h1>

      <div className="flex flex-col gap-6 md:flex-row">
        <SettingsNav />
        <div className="flex-1 min-w-0 space-y-6">
          <Routes>
            <Route index element={<Navigate to="/settings/profile" replace />} />
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
