import { Clock, Loader2 } from "lucide-react";
import { useDataRetention } from "@/hooks/useDataRetention";
import { Button } from "@/components/ui/button";
import { FormError } from "@/components/FormError";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

export function DataRetentionSection() {
  const { dataRetention, isLoading, error, refetch } = useDataRetention();

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <Clock className="text-muted-foreground size-5" />
          <CardTitle>Data Retention</CardTitle>
        </div>
        <CardDescription>
          How long audit log events are kept on this server (configured by the
          operator via <code className="font-mono text-xs">AUDIT_RETENTION_DAYS</code>
          , default 90 days).
        </CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div
            className="flex items-center justify-center py-8"
            role="status"
            aria-label="Loading data retention policy"
          >
            <Loader2 className="text-muted-foreground size-5 animate-spin" />
          </div>
        ) : error ? (
          <div className="space-y-3">
            <FormError error="Failed to load data retention policy." />
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              Try Again
            </Button>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="space-y-3 rounded-lg border p-4">
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">Audit log retention</span>
                <span className="text-sm text-muted-foreground">
                  {dataRetention?.effective_retention_days ??
                    dataRetention?.audit_retention_days ??
                    90}{" "}
                  days
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">Account data</span>
                <span className="text-sm text-muted-foreground">
                  Until account deletion
                </span>
              </div>
            </div>
            <p className="text-muted-foreground text-xs">
              Audit events older than the retention window are deleted automatically.
              Profile and credential data stay until you delete your account.
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
