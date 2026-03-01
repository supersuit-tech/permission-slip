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
          How long your data is kept based on your current plan.
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
            <div className="rounded-lg border p-4 space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">Current Plan</span>
                <span className="text-sm text-muted-foreground">
                  {dataRetention?.plan_name ?? "Free"}
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">Audit Log Retention</span>
                <span className="text-sm text-muted-foreground">
                  {dataRetention?.effective_retention_days ?? dataRetention?.audit_retention_days ?? 7} days
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">Account Data</span>
                <span className="text-sm text-muted-foreground">
                  Until account deletion
                </span>
              </div>
            </div>
            {dataRetention?.grace_period_ends_at ? (
              <p className="text-amber-700 dark:text-amber-300 text-xs">
                Your 90-day audit history is temporarily preserved.
                After {new Date(dataRetention.grace_period_ends_at).toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" })},
                retention will be reduced to {dataRetention.audit_retention_days} days.
              </p>
            ) : (
              <p className="text-xs text-muted-foreground">
                Audit log events older than your retention period are automatically
                deleted. Upgrade to a paid plan for 90-day retention.
              </p>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
