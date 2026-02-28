import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "@/components/ui/card";
import type { ConnectorDetailResponse } from "@/hooks/useConnectorDetail";

interface ConnectorOverviewSectionProps {
  connector: ConnectorDetailResponse;
  enabledAt?: string;
}

export function ConnectorOverviewSection({
  connector,
  enabledAt,
}: ConnectorOverviewSectionProps) {
  const authTypes = connector.required_credentials.map((c) => c.auth_type);
  const uniqueAuthTypes = [...new Set(authTypes)];

  return (
    <Card>
      <CardHeader>
        <CardTitle>{connector.name}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {connector.description && (
          <p className="text-muted-foreground text-sm">
            {connector.description}
          </p>
        )}
        <dl className="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
          {uniqueAuthTypes.length > 0 && (
            <>
              <dt className="text-muted-foreground">Auth type</dt>
              <dd className="font-medium">{uniqueAuthTypes.join(", ")}</dd>
            </>
          )}
          {enabledAt && (
            <>
              <dt className="text-muted-foreground">Enabled since</dt>
              <dd className="font-medium">
                {new Date(enabledAt).toLocaleDateString()}
              </dd>
            </>
          )}
        </dl>
      </CardContent>
    </Card>
  );
}
