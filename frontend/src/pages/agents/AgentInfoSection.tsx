import { useState } from "react";
import { Check, Pencil, X, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { AgentStatusBadge } from "@/components/AgentStatusBadge";
import { formatRelativeTime } from "@/lib/utils";
import { getAgentDisplayName } from "@/lib/agents";
import { useUpdateAgent } from "@/hooks/useUpdateAgent";
import type { AgentDetail } from "@/hooks/useAgent";
import validation from "@/lib/validation";

export function AgentInfoSection({ agent }: { agent: AgentDetail }) {
  const [isEditing, setIsEditing] = useState(false);
  const [editName, setEditName] = useState("");
  const { updateAgent, isLoading } = useUpdateAgent();

  const agentName = getAgentDisplayName(agent);

  function startEditing() {
    setEditName(agentName);
    setIsEditing(true);
  }

  function cancelEditing() {
    setIsEditing(false);
    setEditName("");
  }

  async function saveName() {
    const trimmed = editName.trim();
    if (!trimmed) {
      toast.error("Name cannot be empty");
      return;
    }
    if (trimmed === agentName) {
      cancelEditing();
      return;
    }

    try {
      await updateAgent({
        agentId: agent.agent_id,
        metadata: { name: trimmed },
      });
      toast.success("Agent name updated");
      setIsEditing(false);
    } catch {
      toast.error("Failed to update agent name");
    }
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-3">
          <CardTitle className="flex items-center gap-2">
            {isEditing ? (
              <div className="flex items-center gap-2">
                <Input
                  value={editName}
                  onChange={(e) => setEditName(e.target.value)}
                  aria-label="Agent name"
                  className="h-8 w-60"
                  maxLength={validation.agentName.maxLength}
                  autoFocus
                  disabled={isLoading}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") saveName();
                    if (e.key === "Escape") cancelEditing();
                  }}
                />
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={saveName}
                  disabled={isLoading}
                  aria-label="Save name"
                >
                  {isLoading ? (
                    <Loader2 className="size-4 animate-spin" />
                  ) : (
                    <Check className="size-4" />
                  )}
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={cancelEditing}
                  disabled={isLoading}
                  aria-label="Cancel editing"
                >
                  <X className="size-4" />
                </Button>
              </div>
            ) : (
              <>
                {agentName}
                {agent.status !== "deactivated" && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={startEditing}
                    aria-label="Edit name"
                  >
                    <Pencil className="size-4" />
                  </Button>
                )}
              </>
            )}
          </CardTitle>
          <AgentStatusBadge status={agent.status} />
        </div>
      </CardHeader>
      <CardContent>
        <div className="text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
          <span>
            Registered{" "}
            {agent.registered_at
              ? new Date(agent.registered_at).toLocaleDateString()
              : "—"}
          </span>
          <span aria-hidden="true">·</span>
          <span>Active {formatRelativeTime(agent.last_active_at)}</span>
        </div>
      </CardContent>
    </Card>
  );
}
