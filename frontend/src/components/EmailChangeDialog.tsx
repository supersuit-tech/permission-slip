import { useState, type FormEvent } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { useAuth } from "@/auth/AuthContext";
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

interface EmailChangeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function EmailChangeDialog({
  open,
  onOpenChange,
}: EmailChangeDialogProps) {
  const { user, updateEmail } = useAuth();
  const [newEmail, setNewEmail] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitted, setSubmitted] = useState(false);

  function handleClose(nextOpen: boolean) {
    if (!nextOpen) {
      setNewEmail("");
      setIsSubmitting(false);
      setSubmitted(false);
    }
    onOpenChange(nextOpen);
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();

    const trimmed = newEmail.trim();
    if (!trimmed) return;

    if (trimmed.toLowerCase() === user?.email?.toLowerCase()) {
      toast.error("That's already your current email address.");
      return;
    }

    setIsSubmitting(true);
    const { error } = await updateEmail(trimmed);
    setIsSubmitting(false);

    if (error) {
      console.error("Email change failed:", error);
      toast.error(error.message || "Failed to update email. Please try again.");
      return;
    }

    setSubmitted(true);
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Change Email Address</DialogTitle>
          <DialogDescription>
            {submitted
              ? "Check your inbox to confirm the change."
              : "Enter your new email address. You'll receive a confirmation email to verify the change."}
          </DialogDescription>
        </DialogHeader>

        {submitted ? (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground">
              A confirmation link has been sent to{" "}
              <span className="font-medium text-foreground">{newEmail.trim()}</span>.
              Your email won't change until you confirm.
            </p>
            <DialogFooter>
              <Button variant="secondary" onClick={() => handleClose(false)}>
                Done
              </Button>
            </DialogFooter>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="current-email">Current email</Label>
              <Input
                id="current-email"
                type="email"
                value={user?.email ?? ""}
                disabled
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="new-email">New email</Label>
              <Input
                id="new-email"
                type="email"
                placeholder="you@example.com"
                value={newEmail}
                onChange={(e) => setNewEmail(e.target.value)}
                required
                autoFocus
              />
            </div>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => handleClose(false)}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={isSubmitting || !newEmail.trim()}>
                {isSubmitting && <Loader2 className="animate-spin" />}
                Send Confirmation
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
