import { PolicyLayout } from "./PolicyLayout";

export function PrivacyPolicyPage() {
  return (
    <PolicyLayout title="Privacy Policy">
      <p className="text-muted-foreground">
        Our privacy policy is coming soon. This page will detail what data we
        collect, how we use it, who we share it with, retention periods, and your
        rights under GDPR, CCPA, and other applicable regulations.
      </p>
    </PolicyLayout>
  );
}
