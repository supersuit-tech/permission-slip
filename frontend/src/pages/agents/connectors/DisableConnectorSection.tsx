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

interface DisableConnectorSectionProps {
  agentId: number;
  connectorId: string;
  connectorName: string;
}

export function DisableConnectorSection({
  agentId,
  connectorId,
  connectorName,
}: DisableConnectorSectionProps) {
  const [confirmOpen, setConfirmOpen] = useState(false);
  const { disableConnector, isLoading } = useDisableAgentConnector();
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
      setConfirmOpen(false);
      navigate(`/agents/${agentId}`);
    } catch {
      toast.error("Failed to disable connector");
    }
  }

  return (
    <>
      <Card className="border-destructive/50">
        <CardHeader>
          <CardTitle className="text-destructive">Danger Zone</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">Disable this connector</p>
              <p className="text-muted-foreground text-xs">
                This will prevent the agent from submitting actions from{" "}
                {connectorName}. Any active standing approvals for this
                connector will be revoked.
              </p>
            </div>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => setConfirmOpen(true)}
            >
              Disable
            </Button>
          </div>
        </CardContent>
      </Card>

      <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Disable {connectorName}</DialogTitle>
            <DialogDescription>
              This will disable the <strong>{connectorName}</strong> connector
              for this agent. Any active standing approvals for actions from
              this connector will be automatically revoked.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="secondary"
              onClick={() => setConfirmOpen(false)}
              disabled={isLoading}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDisable}
              disabled={isLoading}
            >
              {isLoading && <Loader2 className="animate-spin" />}
              Disable
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
