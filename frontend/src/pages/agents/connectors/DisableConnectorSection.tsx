import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useDisableAgentConnector } from "@/hooks/useDisableAgentConnector";
import { useDisconnectOAuth } from "@/hooks/useDisconnectOAuth";
import { providerLabel } from "@/lib/labels";

interface DisableConnectorSectionProps {
  agentId: number;
  connectorId: string;
  connectorName: string;
  /** OAuth provider ID — when set, shows a "Remove" option that also disconnects OAuth. */
  oauthProvider?: string;
}

export function DisableConnectorSection({
  agentId,
  connectorId,
  connectorName,
  oauthProvider,
}: DisableConnectorSectionProps) {
  const [disableConfirmOpen, setDisableConfirmOpen] = useState(false);
  const [removeConfirmOpen, setRemoveConfirmOpen] = useState(false);
  const { disableConnector, isLoading: isDisabling } =
    useDisableAgentConnector();
  const { disconnect, isLoading: isDisconnecting } = useDisconnectOAuth();
  const navigate = useNavigate();

  const isLoading = isDisabling || isDisconnecting;

  async function handleDisable() {
    try {
      const result = await disableConnector({ agentId, connectorId });
      const revoked = result.revoked_standing_approvals;
      if (revoked > 0) {
        toast.success(
          `Connector disabled. ${revoked} standing approval${revoked === 1 ? "" : "s"} revoked.`,
        );
      } else {
        toast.success("Connector disabled");
      }
      setDisableConfirmOpen(false);
      navigate(`/agents/${agentId}`);
    } catch {
      toast.error("Failed to disable connector");
    }
  }

  async function handleRemove() {
    // Step 1: disable the connector
    let disableResult: Awaited<ReturnType<typeof disableConnector>>;
    try {
      disableResult = await disableConnector({ agentId, connectorId });
    } catch {
      toast.error("Failed to disable connector");
      return;
    }

    // Step 2: disconnect OAuth (best-effort — connector is already disabled).
    // The Remove button is only rendered when oauthProvider is set, so this
    // branch is always taken. The assertion guards against future refactors.
    if (!oauthProvider) {
      throw new Error("handleRemove called without oauthProvider");
    }
    const revoked = disableResult.revoked_standing_approvals;
    const revokedSuffix =
      revoked > 0
        ? ` ${revoked} standing approval${revoked === 1 ? "" : "s"} revoked.`
        : "";
    try {
      await disconnect(oauthProvider);
      toast.success(
        `Connector removed and OAuth disconnected.${revokedSuffix}`,
      );
    } catch {
      toast.warning(
        `Connector disabled, but OAuth disconnect failed.${revokedSuffix} You can disconnect manually from Settings.`,
      );
    }

    setRemoveConfirmOpen(false);
    navigate(`/agents/${agentId}`);
  }

  return (
    <>
      <Card className="border-destructive/50">
        <CardHeader>
          <CardTitle className="text-destructive">Danger Zone</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">Disable this connector</p>
              <p className="text-muted-foreground text-xs">
                Temporarily prevent the agent from using {connectorName}. Your
                credentials and OAuth connections are preserved &mdash; you can
                re-enable later without reconnecting.
              </p>
            </div>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => setDisableConfirmOpen(true)}
            >
              Disable
            </Button>
          </div>

          {oauthProvider && (
            <>
              <div className="border-t" />
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium">
                    Remove this connector
                  </p>
                  <p className="text-muted-foreground text-xs">
                    Disable the connector and disconnect the{" "}
                    {providerLabel(oauthProvider)} OAuth connection. You will
                    need to re-authorize when re-enabling.
                  </p>
                </div>
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => setRemoveConfirmOpen(true)}
                >
                  Remove
                </Button>
              </div>
            </>
          )}
        </CardContent>
      </Card>

      {/* Disable confirmation */}
      <Dialog open={disableConfirmOpen} onOpenChange={setDisableConfirmOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Disable {connectorName}</DialogTitle>
            <DialogDescription>
              This will disable the <strong>{connectorName}</strong> connector
              for this agent. Any active standing approvals for actions from
              this connector will be automatically revoked.
              {oauthProvider && " Your OAuth connection is preserved."}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="secondary"
              onClick={() => setDisableConfirmOpen(false)}
              disabled={isLoading}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDisable}
              disabled={isLoading}
            >
              {isDisabling && <Loader2 className="animate-spin" />}
              Disable
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Remove confirmation */}
      <Dialog open={removeConfirmOpen} onOpenChange={setRemoveConfirmOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Remove {connectorName}</DialogTitle>
            <DialogDescription>
              This will disable the <strong>{connectorName}</strong> connector
              and disconnect the{" "}
              <strong>{providerLabel(oauthProvider ?? "")}</strong> OAuth
              connection. Standing approvals will be revoked and you will need
              to re-authorize OAuth when re-enabling.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="secondary"
              onClick={() => setRemoveConfirmOpen(false)}
              disabled={isLoading}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleRemove}
              disabled={isLoading}
            >
              {isLoading && <Loader2 className="animate-spin" />}
              Remove
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
