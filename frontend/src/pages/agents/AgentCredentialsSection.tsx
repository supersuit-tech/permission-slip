import { useNavigate } from "react-router-dom";
import { CheckCircle2, Circle, Key, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "@/components/ui/card";
import { useCredentials } from "@/hooks/useCredentials";
import type { AgentConnector } from "@/hooks/useAgentConnectors";

interface AgentCredentialsSectionProps {
  agentId: number;
  connectors: AgentConnector[];
  isLoading: boolean;
  error: string | null;
}

export function AgentCredentialsSection({
  agentId,
  connectors,
  isLoading: connectorsLoading,
  error: connectorsError,
}: AgentCredentialsSectionProps) {
  const navigate = useNavigate();

  // Gather all required credential services across enabled connectors
  const requiredServices = [
    ...new Set(connectors.flatMap((c) => c.required_credentials)),
  ];

  const {
    credentials,
    isLoading: credentialsLoading,
    error: credentialsError,
  } = useCredentials({ enabled: requiredServices.length > 0 });

  const storedServices = new Set(credentials.map((c) => c.service));

  const isLoading = connectorsLoading || credentialsLoading;
  const error = connectorsError ?? credentialsError;

  const connectedCount = requiredServices.filter((s) =>
    storedServices.has(s),
  ).length;
  const totalCount = requiredServices.length;

  // Map service → connector ID for navigation
  const serviceToConnector = new Map<string, string>();
  for (const connector of connectors) {
    for (const service of connector.required_credentials) {
      if (!serviceToConnector.has(service)) {
        serviceToConnector.set(service, connector.id);
      }
    }
  }

  return (
    <Card>
      <CardHeader className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <CardTitle>Credentials</CardTitle>
        {!isLoading && !error && totalCount > 0 && (
          <span className="text-muted-foreground text-sm">
            {connectedCount} of {totalCount} connected
          </span>
        )}
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div
            className="flex items-center justify-center py-8"
            role="status"
            aria-label="Loading credentials"
          >
            <Loader2
              className="text-muted-foreground size-6 animate-spin"
              aria-hidden="true"
            />
          </div>
        ) : error ? (
          <p className="text-destructive text-sm">{error}</p>
        ) : connectors.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <Key className="text-muted-foreground mb-3 size-10" />
            <p className="text-muted-foreground text-sm">
              No connectors enabled — add connectors above to manage credentials.
            </p>
          </div>
        ) : requiredServices.length === 0 ? (
          <p className="text-muted-foreground py-4 text-center text-sm">
            Enabled connectors do not require any credentials.
          </p>
        ) : (
          <div className="space-y-2">
            {requiredServices.map((service) => {
              const isConnected = storedServices.has(service);
              const connectorId = serviceToConnector.get(service);
              return (
                <div
                  key={service}
                  className="flex flex-col gap-2 rounded-lg border p-3 sm:flex-row sm:items-center sm:justify-between"
                >
                  <div className="flex items-center gap-3">
                    {isConnected ? (
                      <CheckCircle2 className="size-5 shrink-0 text-green-600 dark:text-green-400" />
                    ) : (
                      <Circle className="text-muted-foreground size-5 shrink-0" />
                    )}
                    <p className="text-sm font-medium">{service}</p>
                  </div>
                  <div className="flex shrink-0 items-center gap-2 pl-8 sm:pl-0">
                    <span
                      className={`text-xs font-medium ${
                        isConnected
                          ? "text-green-600 dark:text-green-400"
                          : "text-muted-foreground"
                      }`}
                    >
                      {isConnected ? "Connected" : "Needs credentials"}
                    </span>
                    {connectorId && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() =>
                          navigate(
                            `/agents/${agentId}/connectors/${connectorId}`,
                          )
                        }
                      >
                        {isConnected ? "Manage" : "Connect"}
                      </Button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
