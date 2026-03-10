import { ConnectedAccountsSection } from "./ConnectedAccountsSection";
import { OAuthProviderSection } from "./OAuthProviderSection";
import { CredentialSection } from "./CredentialSection";

export function IntegrationsPage() {
  return (
    <>
      <ConnectedAccountsSection />
      <OAuthProviderSection />
      <CredentialSection />
    </>
  );
}
