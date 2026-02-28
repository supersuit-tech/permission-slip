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
import { generateInviteInstructions } from "./agentInstructions";
import { InstructionsBlock } from "./InstructionsBlock";

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
      <DialogContent className={invite ? "sm:max-w-2xl" : undefined}>
        <DialogHeader>
          <DialogTitle>Add an Agent</DialogTitle>
          <DialogDescription>
            {invite
              ? "Copy these instructions and send them to your agent. They contain everything the agent needs to register with Permission Slip."
              : "Generate registration instructions to share with an AI agent. The invite is single-use and expires in 15 minutes."}
          </DialogDescription>
        </DialogHeader>

        {invite ? (
          <InviteInstructionsDisplay invite={invite} />
        ) : (
          <DialogFooter>
            <Button onClick={handleGenerate} disabled={isCreating}>
              {isCreating && <Loader2 className="animate-spin" />}
              Generate Invite Instructions
            </Button>
          </DialogFooter>
        )}
      </DialogContent>
    </Dialog>
  );
}

function InviteInstructionsDisplay({ invite }: { invite: InviteResponse }) {
  const origin = window.location.origin;
  const instructions = generateInviteInstructions(invite.invite_code, origin);

  return (
    <div className="space-y-3">
      <InstructionsBlock instructions={instructions} />
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
