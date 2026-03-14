import { type FormEvent, useState } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useCreateStandingApproval } from "@/hooks/useCreateStandingApproval";
import type { Agent } from "@/hooks/useAgents";

interface CreateStandingApprovalDialogProps {
  agents: Agent[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

function defaultExpiresAt(): string {
  const d = new Date();
  d.setDate(d.getDate() + 30);
  const local = new Date(d.getTime() - d.getTimezoneOffset() * 60000);
  return local.toISOString().slice(0, 16);
}

export function CreateStandingApprovalDialog({
  agents,
  open,
  onOpenChange,
}: CreateStandingApprovalDialogProps) {
  const { createStandingApproval, isPending } = useCreateStandingApproval();

  const [agentId, setAgentId] = useState<number | "">("");
  const [actionType, setActionType] = useState("");
  const [actionVersion, setActionVersion] = useState("1");
  const [constraintsJson, setConstraintsJson] = useState("");
  const [maxExecutions, setMaxExecutions] = useState("");
  const [expiresAt, setExpiresAt] = useState(defaultExpiresAt);

  const activeAgents = agents.filter((a) => a.status !== "deactivated");

  function resetForm() {
    setAgentId("");
    setActionType("");
    setActionVersion("1");
    setConstraintsJson("");
    setMaxExecutions("");
    setExpiresAt(defaultExpiresAt());
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();

    if (!agentId || !actionType || !expiresAt) {
      toast.error("Please fill in all required fields");
      return;
    }

    let constraints: Record<string, unknown>;
    try {
      constraints = JSON.parse(constraintsJson) as Record<string, unknown>;
    } catch {
      toast.error("Constraints must be valid JSON");
      return;
    }

    if (constraints === null || typeof constraints !== "object" || Array.isArray(constraints)) {
      toast.error("Constraints must be a JSON object");
      return;
    }

    try {
      await createStandingApproval({
        agent_id: agentId,
        action_type: actionType,
        action_version: actionVersion || "1",
        constraints,
        max_executions: maxExecutions ? Number(maxExecutions) : null,
        expires_at: new Date(expiresAt).toISOString(),
      });
      toast.success("Standing approval created");
      resetForm();
      onOpenChange(false);
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : "Failed to create standing approval",
      );
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        if (!v) resetForm();
        onOpenChange(v);
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Create Standing Approval</DialogTitle>
          <DialogDescription>
            Pre-authorize an agent to execute an action type without per-request
            approval.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="sa-agent">Agent</Label>
            <select
              id="sa-agent"
              value={agentId}
              onChange={(e) => setAgentId(e.target.value === "" ? "" : Number(e.target.value))}
              className="border-input bg-background ring-offset-background focus-visible:ring-ring flex h-9 w-full rounded-md border px-3 py-1 text-sm focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
              required
            >
              <option value="">Select an agent</option>
              {activeAgents.map((a) => (
                <option key={a.agent_id} value={a.agent_id}>
                  {a.agent_id}
                </option>
              ))}
            </select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="sa-action-type">Action Type</Label>
            <Input
              id="sa-action-type"
              placeholder="e.g. email.send"
              value={actionType}
              onChange={(e) => setActionType(e.target.value)}
              required
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="sa-constraints">Constraints (JSON)</Label>
            <Input
              id="sa-constraints"
              placeholder='{"recipient_pattern": "*@mycompany.com"}'
              value={constraintsJson}
              onChange={(e) => setConstraintsJson(e.target.value)}
              required
            />
            <p className="text-muted-foreground text-xs">
              Parameter constraints for this standing approval. At least one
              non-wildcard constraint is required.
            </p>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="sa-action-version">Version</Label>
              <Input
                id="sa-action-version"
                placeholder="1"
                value={actionVersion}
                onChange={(e) => setActionVersion(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="sa-max-executions">Max Executions</Label>
              <Input
                id="sa-max-executions"
                type="number"
                min="1"
                step="1"
                placeholder="Unlimited"
                value={maxExecutions}
                onChange={(e) => {
                  const value = e.target.value;
                  if (value === "") {
                    setMaxExecutions("");
                    return;
                  }
                  const intValue = parseInt(value, 10);
                  if (Number.isNaN(intValue) || intValue < 1) {
                    return;
                  }
                  setMaxExecutions(String(intValue));
                }}
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="sa-expires-at">Expires At</Label>
            <Input
              id="sa-expires-at"
              type="datetime-local"
              value={expiresAt}
              onChange={(e) => setExpiresAt(e.target.value)}
              required
            />
            <p className="text-muted-foreground text-xs">
              Maximum 90 days from now.
            </p>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="secondary"
              onClick={() => {
                resetForm();
                onOpenChange(false);
              }}
              disabled={isPending}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isPending}>
              {isPending && <Loader2 className="animate-spin" />}
              Create
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
