import { Shield, ShieldAlert, ShieldCheck } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "@/components/ui/card";
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "@/components/ui/table";
import type { ConnectorAction } from "@/hooks/useConnectorDetail";

interface ConnectorActionsSectionProps {
  actions: ConnectorAction[];
}

export function ConnectorActionsSection({
  actions,
}: ConnectorActionsSectionProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Actions</CardTitle>
      </CardHeader>
      <CardContent>
        {actions.length === 0 ? (
          <p className="text-muted-foreground py-4 text-center text-sm">
            No actions defined for this connector.
          </p>
        ) : (
          <div className="overflow-hidden rounded-lg">
            <Table>
              <TableHeader>
                <TableRow className="border-none bg-primary hover:bg-primary">
                  <TableHead className="font-semibold text-primary-foreground">
                    Action
                  </TableHead>
                  <TableHead className="font-semibold text-primary-foreground">
                    Risk
                  </TableHead>
                  <TableHead className="font-semibold text-primary-foreground">
                    Parameters
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody className="[&>tr:nth-child(even)]:bg-muted">
                {actions.map((action) => (
                  <ActionRow key={action.action_type} action={action} />
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function ActionRow({ action }: { action: ConnectorAction }) {
  const params = action.parameters_schema;
  const requiredFields =
    params && "required" in params && Array.isArray(params.required)
      ? (params.required as string[])
      : [];
  const properties =
    params && "properties" in params && params.properties
      ? Object.keys(params.properties as Record<string, unknown>)
      : [];

  return (
    <TableRow>
      <TableCell>
        <div>
          <p className="font-medium">{action.name}</p>
          {action.description && (
            <p className="text-muted-foreground text-xs">
              {action.description}
            </p>
          )}
          <p className="text-muted-foreground mt-0.5 font-mono text-xs">
            {action.action_type}
          </p>
        </div>
      </TableCell>
      <TableCell>
        <RiskBadge level={action.risk_level} />
      </TableCell>
      <TableCell className="text-sm">
        {properties.length > 0 ? (
          <span className="text-muted-foreground">
            {properties.join(", ")}
            {requiredFields.length > 0 && (
              <span className="ml-1 text-xs">
                ({requiredFields.length} required)
              </span>
            )}
          </span>
        ) : (
          <span className="text-muted-foreground">None</span>
        )}
      </TableCell>
    </TableRow>
  );
}

function RiskBadge({ level }: { level?: "low" | "medium" | "high" }) {
  switch (level) {
    case "low":
      return (
        <Badge
          variant="secondary"
          className="bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
        >
          <ShieldCheck className="size-3" />
          Low
        </Badge>
      );
    case "medium":
      return (
        <Badge
          variant="secondary"
          className="bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400"
        >
          <Shield className="size-3" />
          Medium
        </Badge>
      );
    case "high":
      return (
        <Badge variant="destructive">
          <ShieldAlert className="size-3" />
          High
        </Badge>
      );
    default:
      return (
        <Badge variant="outline">
          <Shield className="size-3" />
          Unknown
        </Badge>
      );
  }
}
