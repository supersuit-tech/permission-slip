import { useState } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useCreateInvite, type InviteResponse } from "@/hooks/useCreateInvite";
import { InstructionsBlock } from "./InstructionsBlock";
import { generateInviteInstructions } from "./agentInstructions";

interface InviteCodeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function InviteCodeDialog({
  open,
  onOpenChange,
}: InviteCodeDialogProps) {
  const { createInvite, isLoading: isCreating } = useCreateInvite();
  const [invite, setInvite] = useState<InviteResponse | null>(null);

  async function handleGenerate() {
    try {
      const result = await createInvite();
      setInvite(result);
    } catch {
      toast.error("Failed to generate invite");
    }
  }

  function handleClose(nextOpen: boolean) {
    if (!nextOpen) {
      setInvite(null);
    }
    onOpenChange(nextOpen);
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add an Agent</DialogTitle>
          <DialogDescription>
            {invite
              ? "Tell your agent to run this command. It handles key generation and registration automatically."
              : "Generate a one-time invite command to share with your AI agent. Expires in 15 minutes."}
          </DialogDescription>
        </DialogHeader>

        {invite ? (
          <InviteCommandDisplay invite={invite} />
        ) : (
          <DialogFooter>
            <Button onClick={handleGenerate} disabled={isCreating}>
              {isCreating && <Loader2 className="animate-spin" />}
              Generate Invite Command
            </Button>
          </DialogFooter>
        )}
      </DialogContent>
    </Dialog>
  );
}

function InviteCommandDisplay({ invite }: { invite: InviteResponse }) {
  const origin = window.location.origin;
  const instructions = generateInviteInstructions(invite.invite_code, origin);

  return (
    <div className="space-y-3">
      <InstructionsBlock
        instructions={instructions}
        buttonLabel="Copy Command"
      />
      <p className="text-muted-foreground text-xs">
        Invite expires at{" "}
        {new Date(invite.expires_at).toLocaleString(undefined, {
          dateStyle: "medium",
          timeStyle: "short",
        })}
      </p>
    </div>
  );
}
