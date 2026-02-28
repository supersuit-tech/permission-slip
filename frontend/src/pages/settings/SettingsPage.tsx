import { Link } from "react-router-dom";
import { ArrowLeft } from "lucide-react";
import { AccountSection } from "./AccountSection";
import { NotificationSection } from "./NotificationSection";
import { SecuritySection } from "./SecuritySection";
import { CredentialSection } from "./CredentialSection";

export function SettingsPage() {
  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Link
          to="/"
          className="text-muted-foreground hover:text-foreground transition-colors"
          aria-label="Back to Dashboard"
        >
          <ArrowLeft className="size-5" />
        </Link>
        <h1 className="text-2xl font-bold tracking-tight">Settings</h1>
      </div>

      <AccountSection />
      <SecuritySection />
      <NotificationSection />
      <CredentialSection />
    </div>
  );
}
