import { useState, type FormEvent } from "react";
import { AlertTriangle, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { useAuth } from "@/auth/AuthContext";
import { useDeleteAccount } from "@/hooks/useDeleteAccount";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

export function DangerZoneSection() {
  const [dialogOpen, setDialogOpen] = useState(false);

  return (
    <>
      <Card className="border-destructive/50">
        <CardHeader>
          <div className="flex items-center gap-2">
            <AlertTriangle className="text-destructive size-5" />
            <CardTitle className="text-destructive">Danger Zone</CardTitle>
          </div>
          <CardDescription>
            Irreversible actions that permanently affect your account.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between rounded-lg border border-destructive/30 p-4">
            <div className="space-y-0.5">
              <p className="text-sm font-medium">Delete Account</p>
              <p className="text-xs text-muted-foreground">
                Permanently delete your account and all associated data.
              </p>
            </div>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => setDialogOpen(true)}
            >
              Delete Account
            </Button>
          </div>
        </CardContent>
      </Card>

      <DeleteAccountDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
      />
    </>
  );
}

function DeleteAccountDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { signOut } = useAuth();
  const { deleteAccount, isDeleting } = useDeleteAccount();
  const [confirmText, setConfirmText] = useState("");

  const isConfirmed = confirmText === "DELETE";

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!isConfirmed) return;

    try {
      await deleteAccount();
      toast.success("Account deleted successfully.");
      // Sign out after deletion — clears the session.
      await signOut();
    } catch {
      toast.error("Failed to delete account. Please try again.");
    }
  }

  function handleOpenChange(nextOpen: boolean) {
    if (!isDeleting) {
      setConfirmText("");
      onOpenChange(nextOpen);
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Are you sure?</DialogTitle>
          <DialogDescription>
            This action is <strong className="text-foreground">permanent and irreversible</strong>.
            Deleting your account will immediately remove:
          </DialogDescription>
        </DialogHeader>

        <ul className="text-sm text-muted-foreground list-disc pl-5 space-y-1">
          <li>Your profile and contact information</li>
          <li>All registered agents</li>
          <li>All stored credentials</li>
          <li>All approval history and audit logs</li>
          <li>All standing approvals</li>
          <li>Your subscription</li>
        </ul>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="delete-confirm">
              Type <span className="font-mono font-bold">DELETE</span> to confirm
            </Label>
            <Input
              id="delete-confirm"
              value={confirmText}
              onChange={(e) => setConfirmText(e.target.value)}
              placeholder="DELETE"
              autoComplete="off"
              disabled={isDeleting}
            />
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => handleOpenChange(false)}
              disabled={isDeleting}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              variant="destructive"
              disabled={!isConfirmed || isDeleting}
            >
              {isDeleting && <Loader2 className="animate-spin" />}
              Delete My Account
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
