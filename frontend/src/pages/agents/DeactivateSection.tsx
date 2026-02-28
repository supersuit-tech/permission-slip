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
import { useDeactivateAgent } from "@/hooks/useDeactivateAgent";

export function DeactivateSection({ agentId }: { agentId: number }) {
  const [confirmOpen, setConfirmOpen] = useState(false);
  const { deactivateAgent, isLoading } = useDeactivateAgent();
  const navigate = useNavigate();

  async function handleDeactivate() {
    try {
      await deactivateAgent(agentId);
      toast.success("Agent deactivated");
      setConfirmOpen(false);
      navigate("/");
    } catch {
      toast.error("Failed to deactivate agent");
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
              <p className="text-sm font-medium">Deactivate this agent</p>
              <p className="text-muted-foreground text-xs">
                This will revoke all standing approvals and prevent the agent
                from making further requests. This action cannot be undone.
              </p>
            </div>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => setConfirmOpen(true)}
            >
              Deactivate
            </Button>
          </div>
        </CardContent>
      </Card>

      <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Deactivate Agent</DialogTitle>
            <DialogDescription>
              This will deactivate agent <strong>{agentId}</strong> and revoke
              all its standing approvals. This action cannot be undone.
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
              onClick={handleDeactivate}
              disabled={isLoading}
            >
              {isLoading && <Loader2 className="animate-spin" />}
              Deactivate
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
