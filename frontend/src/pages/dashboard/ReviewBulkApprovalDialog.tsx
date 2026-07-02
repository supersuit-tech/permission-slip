import { useCallback, useEffect, useMemo, useState } from "react";
import { Bot, Check, Loader2, XCircle } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { buildSummary } from "@/components/ActionPreviewSummary";
import { useActionSchema } from "@/hooks/useActionSchema";
import type { ParametersSchema } from "@/lib/parameterSchema";
import { useApprovalBulkGroup } from "@/hooks/useApprovalBulkGroup";
import { useBulkDecideApprovalGroup } from "@/hooks/useBulkDecideApprovalGroup";
import type { ApprovalSummary } from "@/hooks/useApprovals";
import { CountdownBadge, RiskBadge } from "./approval-components";

interface ReviewBulkApprovalDialogProps {
  bulkGroupId: string;
  agentDisplayName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

type ItemDecision = "approve" | "deny";

export function ReviewBulkApprovalDialog({
  bulkGroupId,
  agentDisplayName,
  open,
  onOpenChange,
}: ReviewBulkApprovalDialogProps) {
  const { data: group, isLoading, error } = useApprovalBulkGroup(
    open ? bulkGroupId : undefined,
  );
  const { schema, actionName, displayTemplate } = useActionSchema(
    group?.action_type ?? "",
  );
  const { decideBulkGroup, isPending, result } = useBulkDecideApprovalGroup();

  const [decisions, setDecisions] = useState<Record<string, ItemDecision>>({});
  const [submitted, setSubmitted] = useState(false);

  useEffect(() => {
    if (!group?.items) return;
    setDecisions((prev) => {
      const next = { ...prev };
      for (const item of group.items) {
        if (!(item.approval_id in next)) {
          next[item.approval_id] = "approve";
        }
      }
      return next;
    });
  }, [group?.items]);

  const pendingItems = useMemo(
    () => group?.items.filter((i) => i.status === "pending") ?? [],
    [group?.items],
  );

  const setAll = useCallback((decision: ItemDecision) => {
    setDecisions((prev) => {
      const next = { ...prev };
      for (const item of pendingItems) {
        next[item.approval_id] = decision;
      }
      return next;
    });
  }, [pendingItems]);

  const handleSubmit = async () => {
    if (!group) return;
    try {
      await decideBulkGroup({
        bulkGroupId: group.bulk_group_id,
        decisions: pendingItems.map((item) => ({
          approval_id: item.approval_id,
          decision: decisions[item.approval_id] ?? "approve",
        })),
      });
      setSubmitted(true);
      toast.success("Bulk review submitted");
    } catch {
      toast.error("Failed to submit bulk review");
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] max-w-2xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex flex-wrap items-center gap-2">
            <Bot className="size-5" />
            Bulk approval — {group?.action_type ?? "…"}
            {group && (
              <Badge variant="secondary">{group.item_count} items</Badge>
            )}
          </DialogTitle>
        </DialogHeader>

        {isLoading && (
          <div className="space-y-3">
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-24 w-full" />
          </div>
        )}

        {error && (
          <p className="text-destructive text-sm">Could not load bulk group.</p>
        )}

        {group && !submitted && (
          <>
            <p className="text-muted-foreground text-sm">
              From {agentDisplayName}. Each item defaults to approve — toggle off
              any you want to deny, then submit once.
            </p>
            {group.expires_at && (
              <CountdownBadge expiresAt={group.expires_at} />
            )}

            <div className="flex gap-2">
              <Button type="button" variant="outline" size="sm" onClick={() => setAll("approve")}>
                Approve all
              </Button>
              <Button type="button" variant="outline" size="sm" onClick={() => setAll("deny")}>
                Deny all
              </Button>
            </div>

            <ul className="space-y-3">
              {group.items.map((item) => (
                <BulkItemRow
                  key={item.approval_id}
                  item={item}
                  schema={schema}
                  actionName={actionName}
                  displayTemplate={displayTemplate}
                  decision={decisions[item.approval_id] ?? "approve"}
                  onDecisionChange={(d) =>
                    setDecisions((prev) => ({ ...prev, [item.approval_id]: d }))
                  }
                />
              ))}
            </ul>

            <div className="flex justify-end gap-2 pt-2">
              <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button
                type="button"
                disabled={isPending || pendingItems.length === 0}
                onClick={() => void handleSubmit()}
              >
                {isPending && <Loader2 className="mr-2 size-4 animate-spin" />}
                Submit review
              </Button>
            </div>
          </>
        )}

        {submitted && result && (
          <div className="space-y-3">
            <div className="flex items-center gap-2 text-sm font-medium">
              <Check className="size-4 text-green-600" />
              Review submitted
            </div>
            <ul className="space-y-2 text-sm">
              {result.results.map((r) => (
                <li key={r.approval_id} className="flex items-center gap-2">
                  {r.status === "approved" && <Check className="size-4 text-green-600" />}
                  {r.status === "denied" && <XCircle className="size-4 text-red-600" />}
                  <span className="font-mono text-xs">{r.approval_id}</span>
                  <span>{r.status}</span>
                  {r.execution_status === "error" && (
                    <span className="text-destructive">execution failed</span>
                  )}
                </li>
              ))}
            </ul>
            <Button type="button" onClick={() => onOpenChange(false)}>
              Close
            </Button>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

function BulkItemRow({
  item,
  schema,
  actionName,
  displayTemplate,
  decision,
  onDecisionChange,
}: {
  item: ApprovalSummary;
  schema: ParametersSchema | null;
  actionName: string | null;
  displayTemplate: string | null;
  decision: ItemDecision;
  onDecisionChange: (d: ItemDecision) => void;
}) {
  const summary = buildSummary(
    item.action.type,
    item.action.parameters as Record<string, unknown>,
    schema,
    actionName,
    displayTemplate,
    item.resource_details as Record<string, unknown> | undefined,
  );
  const isPending = item.status === "pending";

  return (
    <li className="rounded-lg border p-3">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1 space-y-1">
          <div className="flex flex-wrap items-center gap-2">
            <RiskBadge level={item.context.risk_level} />
            {!isPending && (
              <Badge variant="outline">{item.status}</Badge>
            )}
          </div>
          <p
            className="text-muted-foreground line-clamp-2 text-xs break-words"
            title={summary}
          >
            {summary}
          </p>
        </div>
        {isPending && (
          <label className="flex shrink-0 cursor-pointer items-center gap-2 text-sm">
            <Checkbox
              checked={decision === "approve"}
              onCheckedChange={(checked) =>
                onDecisionChange(checked ? "approve" : "deny")
              }
            />
            Approve
          </label>
        )}
      </div>
    </li>
  );
}
