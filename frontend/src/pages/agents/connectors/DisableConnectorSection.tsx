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
  /**
   * OAuth provider ID (e.g. "github") from the connector spec.
   * When set, the "Remove" option is shown and its description mentions
   * disconnecting OAuth. Does NOT drive the actual disconnect call —
   * see `oauthConnectionId` for that.
   */
  oauthProvider?: string;
  /**
   * The specific OAuth connection ID currently bound to this connector
   * (e.g. "oconn_abc123"). Passed to `useDisconnectOAuth` on Remove so
   * the correct connection is disconnected. If undefined, the OAuth
   * disconnect step is skipped silently.
   */
  oauthConnectionId?: string;
  /**
   * True when an API key credential is assigned to this connector.
   * When true (and `oauthProvider` is not set), the "Remove" option is
   * shown so the user can permanently delete the saved API key.
   */
  hasApiKeyCredential?: boolean;
}

export function DisableConnectorSection({
  agentId,
  connectorId,
  connectorName,
  oauthProvider,
  oauthConnectionId,
  hasApiKeyCredential = false,
}: DisableConnectorSectionProps) {
  const [disableConfirmOpen, setDisableConfirmOpen] = useState(false);
  const [removeConfirmOpen, setRemoveConfirmOpen] = useState(false);
  const [isRemoving, setIsRemoving] = useState(false);
  const { disableConnector, isLoading: isDisabling } =
    useDisableAgentConnector();
  const { disconnect } = useDisconnectOAuth();
  const navigate = useNavigate();

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
    setIsRemoving(true);
    try {
      // Step 1: disable the connector and delete any API key credentials.
      let disableResult: Awaited<ReturnType<typeof disableConnector>>;
      try {
        disableResult = await disableConnector({ agentId, connectorId, deleteCredentials: true });
      } catch {
        toast.error("Failed to disable connector");
        return;
      }

      const revoked = disableResult.revoked_standing_approvals;
      const revokedSuffix =
        revoked > 0
          ? ` ${revoked} standing approval${revoked === 1 ? "" : "s"} revoked.`
          : "";

      // Step 2: disconnect OAuth (best-effort — connector is already disabled).
      // oauthConnectionId is the specific connection to disconnect; skip if not bound.
      if (oauthConnectionId) {
        try {
          await disconnect(oauthConnectionId);
          toast.success(
            `Connector removed and OAuth disconnected.${revokedSuffix}`,
          );
        } catch {
          toast.warning(
            `Connector disabled and API keys deleted, but OAuth disconnect failed.${revokedSuffix} You can disconnect manually from Settings.`,
          );
        }
      } else {
        toast.success(`Connector removed.${revokedSuffix}`);
      }

      setRemoveConfirmOpen(false);
      navigate(`/agents/${agentId}`);
    } finally {
      setIsRemoving(false);
    }
  }

  return (
    <>
      <Card className="border-destructive/50">
        <CardHeader>
          <CardTitle className="text-destructive">Danger Zone</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="min-w-0">
              <p className="text-sm font-medium">Disable this connector</p>
              <p className="text-muted-foreground text-sm">
                Temporarily prevent the agent from using {connectorName}. Your
                credentials{oauthProvider && " and OAuth connections"} are
                preserved &mdash; you can re-enable later without reconnecting.
              </p>
            </div>
            <Button
              variant="destructive"
              size="sm"
              className="shrink-0 self-start sm:self-center"
              onClick={() => setDisableConfirmOpen(true)}
              disabled={isRemoving}
            >
              Disable
            </Button>
          </div>

          {(oauthProvider || hasApiKeyCredential) && (
            <>
              <div className="border-t" />
              <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div className="min-w-0">
                  <p className="text-sm font-medium">
                    Remove this connector
                  </p>
                  <p className="text-muted-foreground text-sm">
                    {oauthProvider ? (
                      <>
                        Disable the connector, disconnect the{" "}
                        {providerLabel(oauthProvider)} OAuth connection, and
                        permanently delete any saved API keys. You will need to
                        re-authorize and re-add credentials when re-enabling.
                      </>
                    ) : (
                      <>
                        Disable the connector and permanently delete the saved
                        API key. You will need to re-add credentials when
                        re-enabling.
                      </>
                    )}
                  </p>
                </div>
                <Button
                  variant="destructive"
                  size="sm"
                  className="shrink-0 self-start sm:self-center"
                  disabled={isRemoving}
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
              disabled={isDisabling}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDisable}
              disabled={isDisabling}
            >
              {isDisabling && <Loader2 className="animate-spin" />}
              Disable
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Remove confirmation */}
      <Dialog open={removeConfirmOpen} onOpenChange={(open) => !isRemoving && setRemoveConfirmOpen(open)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Remove {connectorName}</DialogTitle>
            <DialogDescription>
              {oauthProvider ? (
                <>
                  This will disable the <strong>{connectorName}</strong>{" "}
                  connector, disconnect the{" "}
                  <strong>{providerLabel(oauthProvider)}</strong> OAuth
                  connection, and permanently delete any saved API keys.
                  Standing approvals will be revoked. You will need to
                  re-authorize and re-add credentials when re-enabling.
                </>
              ) : (
                <>
                  This will disable the <strong>{connectorName}</strong>{" "}
                  connector and permanently delete the saved API key. Standing
                  approvals will be revoked. You will need to re-add credentials
                  when re-enabling.
                </>
              )}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="secondary"
              onClick={() => setRemoveConfirmOpen(false)}
              disabled={isRemoving}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleRemove}
              disabled={isRemoving}
            >
              {isRemoving && <Loader2 className="animate-spin" />}
              Remove
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
