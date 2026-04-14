import { type FormEvent, useEffect, useState } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useCreateStandingApproval } from "@/hooks/useCreateStandingApproval";
import { useUpdateStandingApproval } from "@/hooks/useUpdateStandingApproval";
import { useRevokeStandingApproval } from "@/hooks/useRevokeStandingApproval";
import type { ActionConfiguration } from "@/hooks/useActionConfigs";
import type { StandingApproval } from "@/hooks/useStandingApprovals";
import { StepLimits } from "@/pages/dashboard/StandingApprovalSteps";
import {
  pickPrimaryStandingApproval,
  standingApprovalRowStatus,
} from "@/lib/standingApprovalStatus";

function standingApprovalConstraintsForCreate(
  params: Record<string, unknown>,
): Record<string, unknown> {
  const entries = Object.entries(params);
  if (entries.length === 0) {
    return {};
  }
  const allBareWildcard = entries.every(([, v]) => v === "*");
  if (allBareWildcard) {
    return {};
  }
  return params;
}

function defaultExpiresAtLocal(): string {
  const d = new Date();
  d.setDate(d.getDate() + 30);
  const local = new Date(d.getTime() - d.getTimezoneOffset() * 60000);
  return local.toISOString().slice(0, 16);
}

interface ActionConfigStandingApprovalSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  agentId: number;
  config: ActionConfiguration;
  standingRows: StandingApproval[];
  onSuccess: () => void;
}

export function ActionConfigStandingApprovalSheet({
  open,
  onOpenChange,
  agentId,
  config,
  standingRows,
  onSuccess,
}: ActionConfigStandingApprovalSheetProps) {
  const { createStandingApproval, isPending: isCreatePending } =
    useCreateStandingApproval();
  const { updateStandingApproval, isPending: isUpdatePending } =
    useUpdateStandingApproval();
  const { revokeStandingApproval, isPending: isRevokePending } =
    useRevokeStandingApproval();
  const isPending = isCreatePending || isUpdatePending || isRevokePending;

  const primary = pickPrimaryStandingApproval(standingRows);
  const rowStatus = standingApprovalRowStatus(standingRows);
  const isEditActive = primary?.status === "active";

  const [noExpiry, setNoExpiry] = useState(true);
  const [expiresAt, setExpiresAt] = useState(defaultExpiresAtLocal);

  useEffect(() => {
    if (!open) return;
    if (isEditActive && primary) {
      setNoExpiry(!primary.expires_at);
      if (primary.expires_at) {
        const d = new Date(primary.expires_at);
        const local = new Date(d.getTime() - d.getTimezoneOffset() * 60000);
        setExpiresAt(local.toISOString().slice(0, 16));
      } else {
        setExpiresAt(defaultExpiresAtLocal());
      }
    } else {
      setNoExpiry(true);
      setExpiresAt(defaultExpiresAtLocal());
    }
  }, [open, isEditActive, primary]);

  async function handleRevoke() {
    if (!primary) return;
    try {
      await revokeStandingApproval(primary.standing_approval_id);
      toast.success("Standing approval revoked");
      onSuccess();
      onOpenChange(false);
    } catch (e) {
      toast.error(
        e instanceof Error ? e.message : "Failed to revoke standing approval",
      );
    }
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (config.status !== "active") {
      toast.error(
        "Enable this action configuration before adding a standing approval.",
      );
      return;
    }

    if (!noExpiry && (!expiresAt || Number.isNaN(new Date(expiresAt).getTime()))) {
      toast.error("Please enter a valid expiration date");
      return;
    }

    if (isEditActive && primary) {
      try {
        await updateStandingApproval(primary.standing_approval_id, {
          constraints: primary.constraints as Record<string, unknown>,
          expires_at: noExpiry ? null : new Date(expiresAt).toISOString(),
        });
        toast.success("Standing approval updated");
        onSuccess();
        onOpenChange(false);
      } catch (err) {
        toast.error(
          err instanceof Error
            ? err.message
            : "Failed to update standing approval",
        );
      }
      return;
    }

    const constraints = standingApprovalConstraintsForCreate(
      config.parameters as Record<string, unknown>,
    );

    try {
      await createStandingApproval({
        agent_id: agentId,
        action_type: config.action_type,
        action_version: "1",
        constraints,
        source_action_configuration_id: config.id,
        ...(noExpiry ? {} : { expires_at: new Date(expiresAt).toISOString() }),
      });
      toast.success("Standing approval created");
      onSuccess();
      onOpenChange(false);
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : "Failed to create standing approval",
      );
    }
  }

  const canSubmit =
    !isPending &&
    config.status === "active" &&
    (noExpiry || !!expiresAt);

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="flex w-full flex-col sm:max-w-md">
        <SheetHeader>
          <SheetTitle>Standing approval</SheetTitle>
          <SheetDescription>
            {isEditActive
              ? "Adjust expiration for this configuration."
              : "Auto-approve requests that match this configuration, with optional expiry."}
          </SheetDescription>
        </SheetHeader>
        <form
          onSubmit={handleSubmit}
          className="flex flex-1 flex-col gap-4 overflow-y-auto px-1 py-2"
        >
          {rowStatus !== "none" && !isEditActive && (
            <p className="text-muted-foreground text-xs">
              Current status: {rowStatus}. Create a new standing approval to replace
              the inactive one.
            </p>
          )}
          {config.status !== "active" && (
            <p className="text-muted-foreground text-sm">
              This configuration is disabled. Enable it before creating a standing approval.
            </p>
          )}
          <StepLimits
            expiresAt={expiresAt}
            onExpiresAtChange={setExpiresAt}
            noExpiry={noExpiry}
            onNoExpiryChange={setNoExpiry}
          />
          <SheetFooter className="mt-auto flex flex-col gap-2 px-0 sm:flex-col">
            {isEditActive && (
              <Button
                type="button"
                variant="destructive"
                className="w-full"
                disabled={isPending}
                onClick={() => void handleRevoke()}
              >
                {isRevokePending ? (
                  <Loader2 className="size-4 animate-spin" />
                ) : null}
                Revoke standing approval
              </Button>
            )}
            <div className="flex w-full gap-2">
              <Button
                type="button"
                variant="secondary"
                className="flex-1"
                disabled={isPending}
                onClick={() => onOpenChange(false)}
              >
                Close
              </Button>
              <Button type="submit" className="flex-1" disabled={!canSubmit}>
                {isCreatePending || isUpdatePending ? (
                  <Loader2 className="size-4 animate-spin" />
                ) : null}
                {isEditActive ? "Save" : "Create"}
              </Button>
            </div>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  );
}
