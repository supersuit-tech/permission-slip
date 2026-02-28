import { Clock, Loader2 } from "lucide-react";
import { useDataRetention } from "@/hooks/useDataRetention";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

export function DataRetentionSection() {
  const { dataRetention, isLoading } = useDataRetention();

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
                  {dataRetention?.audit_retention_days ?? 7} days
                </span>
              </div>
            </div>
            <p className="text-xs text-muted-foreground">
              Audit log events older than your retention period are automatically
              deleted. Upgrade to a paid plan for 90-day retention.
            </p>
            <p className="text-xs text-muted-foreground">
              Account data (profile, credentials) is retained until you
              explicitly delete your account.
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
