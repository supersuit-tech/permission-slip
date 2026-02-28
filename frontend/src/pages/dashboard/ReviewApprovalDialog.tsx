import { useState, useCallback } from "react";
import { Loader2, Clock, Shield, Bot, AlertTriangle } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useApproveApproval } from "@/hooks/useApproveApproval";
import { useDenyApproval } from "@/hooks/useDenyApproval";
import { useActionSchema } from "@/hooks/useActionSchema";
import type { ApprovalSummary } from "@/hooks/useApprovals";
import { ActionPreviewSummary } from "@/components/ActionPreviewSummary";
import { SchemaParameterDetails } from "@/components/SchemaParameterDetails";
import {
  useCountdown,
  formatCountdown,
  urgencyColor,
  RiskBadge,
  ConfirmationCodeBanner,
} from "./approval-components";

interface ReviewApprovalDialogProps {
  approval: ApprovalSummary;
  agentDisplayName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ReviewApprovalDialog({
  approval,
  agentDisplayName,
  open,
  onOpenChange,
}: ReviewApprovalDialogProps) {
  const [confirmationCode, setConfirmationCode] = useState<string | null>(null);
  const { approveApproval, isPending: isApproving } = useApproveApproval();
  const { denyApproval, isPending: isDenying } = useDenyApproval();
  const { schema, actionName } = useActionSchema(approval.action.type);
  const remaining = useCountdown(approval.expires_at);
  const isExpired = remaining <= 0;
  const isBusy = isApproving || isDenying;

  const handleApprove = useCallback(async () => {
    try {
      const result = await approveApproval(approval.approval_id);
      setConfirmationCode(result.confirmation_code);
    } catch {
      toast.error("Failed to approve request. Please try again.");
    }
  }, [approveApproval, approval.approval_id]);

  const handleDeny = useCallback(async () => {
    try {
      await denyApproval(approval.approval_id);
      toast.success("Request denied");
      onOpenChange(false);
    } catch {
      toast.error("Failed to deny request. Please try again.");
    }
  }, [denyApproval, approval.approval_id, onOpenChange]);

  function handleClose(nextOpen: boolean) {
    if (!nextOpen) {
      setConfirmationCode(null);
    }
    onOpenChange(nextOpen);
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Review Approval Request</DialogTitle>
        </DialogHeader>

        {confirmationCode ? (
          <div className="space-y-6 py-4">
            <div className="flex flex-col items-center gap-4">
              <div className="rounded-full bg-green-100 p-3 dark:bg-green-900/30">
                <Shield className="size-8 text-green-600 dark:text-green-400" aria-hidden="true" />
              </div>
              <p className="text-lg font-semibold">Request Approved</p>
            </div>
            <ConfirmationCodeBanner code={confirmationCode} copyable />
          </div>
        ) : (
          <div className="space-y-4 sm:space-y-6">
            {/* Agent */}
            <div className="flex items-center gap-3">
              <div className="bg-muted rounded-full p-2">
                <Bot className="text-muted-foreground size-5" aria-hidden="true" />
              </div>
              <div className="min-w-0">
                <p className="truncate text-sm font-medium">{agentDisplayName}</p>
                {agentDisplayName !== String(approval.agent_id) && (
                  <p className="text-muted-foreground truncate font-mono text-xs">
                    {approval.agent_id}
                  </p>
                )}
              </div>
            </div>

            {/* Action & Risk */}
            <div className="bg-muted/50 space-y-3 rounded-lg border p-3 sm:p-4">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div className="flex min-w-0 flex-wrap items-center gap-2">
                  <div className="min-w-0">
                    <span className="block truncate text-sm font-semibold">
                      {actionName ?? approval.action.type}
                    </span>
                    {actionName && (
                      <span className="text-muted-foreground block truncate font-mono text-xs">
                        {approval.action.type}
                      </span>
                    )}
                  </div>
                  <RiskBadge level={approval.context.risk_level} />
                </div>
                <span
                  aria-label={isExpired ? "Request expired" : `Time remaining: ${formatCountdown(remaining)}`}
                  className={`inline-flex shrink-0 items-center gap-1 text-xs font-medium ${urgencyColor(remaining)}`}
                >
                  <Clock className="size-3" aria-hidden="true" />
                  {isExpired ? "Expired" : formatCountdown(remaining)}
                </span>
              </div>

              {approval.context.description && (
                <p className="text-muted-foreground text-sm">
                  {approval.context.description}
                </p>
              )}

              {/* Action preview summary */}
              <div className="border-t pt-3">
                <ActionPreviewSummary
                  actionType={approval.action.type}
                  parameters={approval.action.parameters as Record<string, unknown>}
                  schema={schema}
                  actionName={actionName}
                />
              </div>
            </div>

            {/* Parameters (collapsible — summary above covers the essentials) */}
            {Object.keys(approval.action.parameters).length > 0 && (
              <details className="group">
                <summary className="flex cursor-pointer items-center gap-1 text-sm font-medium select-none">
                  <span className="text-muted-foreground transition-transform group-open:rotate-90" aria-hidden="true">&#9656;</span>
                  Parameters
                </summary>
                <div className="bg-muted/50 mt-2 overflow-x-auto rounded-lg border p-3 sm:p-4">
                  <SchemaParameterDetails
                    parameters={approval.action.parameters as Record<string, unknown>}
                    schema={schema}
                  />
                </div>
              </details>
            )}

            {/* High risk warning */}
            {approval.context.risk_level === "high" && (
              <div role="alert" className="bg-destructive/10 border-destructive/20 flex items-start gap-2 rounded-lg border p-3">
                <AlertTriangle className="text-destructive mt-0.5 size-4 shrink-0" aria-hidden="true" />
                <p className="text-destructive text-sm">
                  This is a high-risk action. Please review the details carefully
                  before approving.
                </p>
              </div>
            )}
          </div>
        )}

        {!confirmationCode && (
          <DialogFooter>
            <Button
              variant="outline"
              size="lg"
              disabled={isBusy || isExpired}
              onClick={handleDeny}
            >
              {isDenying ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                "Deny"
              )}
            </Button>
            <Button
              size="lg"
              disabled={isBusy || isExpired}
              onClick={handleApprove}
            >
              {isApproving ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                "Approve"
              )}
            </Button>
          </DialogFooter>
        )}
      </DialogContent>
    </Dialog>
  );
}
