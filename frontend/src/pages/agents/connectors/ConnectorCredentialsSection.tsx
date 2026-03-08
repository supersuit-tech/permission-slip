import { Loader2 } from "lucide-react";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "@/components/ui/card";
import { useCredentials } from "@/hooks/useCredentials";
import type { CredentialSummary } from "@/hooks/useCredentials";
import { useOAuthConnections } from "@/hooks/useOAuthConnections";
import type { RequiredCredential } from "@/hooks/useConnectorDetail";
import { OAuthCredentialRow } from "./OAuthCredentialRow";
import { CredentialRow } from "./CredentialRow";

interface ConnectorCredentialsSectionProps {
  requiredCredentials: RequiredCredential[];
}

export function ConnectorCredentialsSection({
  requiredCredentials,
}: ConnectorCredentialsSectionProps) {
  const hasRequiredCredentials = requiredCredentials.length > 0;
  const hasOAuth = requiredCredentials.some((c) => c.auth_type === "oauth2");
  const { credentials, isLoading, error } = useCredentials({
    enabled: hasRequiredCredentials,
  });
  const {
    connections,
    isLoading: oauthLoading,
    error: oauthError,
  } = useOAuthConnections();

  const storedByService = new Map<string, CredentialSummary[]>();
  for (const cred of credentials) {
    const list = storedByService.get(cred.service) ?? [];
    list.push(cred);
    storedByService.set(cred.service, list);
  }

  const oauthByProvider = new Map(
    connections.map((c) => [c.provider, c]),
  );

  const loading = isLoading || (hasOAuth && oauthLoading);

  const hasMixedAuth =
    requiredCredentials.length > 1 &&
    requiredCredentials.some((c) => c.auth_type === "oauth2") &&
    requiredCredentials.some((c) => c.auth_type !== "oauth2");

  return (
    <Card>
      <CardHeader>
        <CardTitle>Credentials</CardTitle>
      </CardHeader>
      <CardContent>
        {!hasRequiredCredentials ? (
          <p className="text-muted-foreground py-4 text-center text-sm">
            This connector does not require any credentials.
          </p>
        ) : loading ? (
          <div className="flex items-center justify-center py-4">
            <Loader2
              className="text-muted-foreground size-5 animate-spin"
              aria-hidden="true"
            />
          </div>
        ) : error || (hasOAuth && oauthError) ? (
          <p className="text-destructive text-sm">{error ?? oauthError}</p>
        ) : (
          <div className="space-y-3">
            {requiredCredentials.map((cred, index) => {
              const showDivider = hasMixedAuth && index > 0;
              return (
                <div key={cred.service}>
                  {showDivider && (
                    <div className="flex items-center gap-3 py-1">
                      <div className="bg-border h-px flex-1" />
                      <span className="text-muted-foreground text-xs font-medium uppercase">
                        or
                      </span>
                      <div className="bg-border h-px flex-1" />
                    </div>
                  )}
                  {cred.auth_type === "oauth2" ? (
                    <OAuthCredentialRow
                      requiredCredential={cred}
                      connection={oauthByProvider.get(
                        cred.oauth_provider ?? "",
                      )}
                      recommended={hasMixedAuth}
                    />
                  ) : (
                    <CredentialRow
                      requiredCredential={cred}
                      storedCredentials={
                        storedByService.get(cred.service) ?? []
                      }
                    />
                  )}
                </div>
              );
            })}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
