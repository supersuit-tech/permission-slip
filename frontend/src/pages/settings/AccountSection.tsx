import { useState, useEffect, type FormEvent } from "react";
import { Loader2, User } from "lucide-react";
import { toast } from "sonner";
import { useProfile } from "@/hooks/useProfile";
import { useUpdateProfile } from "@/hooks/useUpdateProfile";
import { useAuth } from "@/auth/AuthContext";
import { EmailChangeDialog } from "@/components/EmailChangeDialog";
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

export function AccountSection() {
  const { user } = useAuth();
  const { profile } = useProfile();
  const { updateProfile, isLoading } = useUpdateProfile();

  const [email, setEmail] = useState("");
  const [phone, setPhone] = useState("");
  const [isDirty, setIsDirty] = useState(false);
  const [emailDialogOpen, setEmailDialogOpen] = useState(false);

  // Sync form fields when profile data loads or changes (e.g. after save).
  useEffect(() => {
    if (profile) {
      setEmail(profile.email ?? "");
      setPhone(profile.phone ?? "");
      setIsDirty(false);
    }
  }, [profile?.email, profile?.phone]);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();

    try {
      await updateProfile({
        email: email.trim() || null,
        phone: phone.trim() || null,
      });
      setIsDirty(false);
      toast.success("Contact information updated.");
    } catch {
      toast.error("Failed to update contact information. Please try again.");
    }
  }

  return (
    <>
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <User className="text-muted-foreground size-5" />
          <CardTitle>Account</CardTitle>
        </div>
        <CardDescription>
          Manage your profile and contact information.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="settings-username">Username</Label>
              <Input
                id="settings-username"
                value={profile?.username ?? ""}
                disabled
              />
              <p className="text-xs text-muted-foreground">
                Your protocol-level username cannot be changed.
              </p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="settings-login-email">Login email</Label>
              <div className="flex gap-2">
                <Input
                  id="settings-login-email"
                  type="email"
                  value={user?.email ?? ""}
                  disabled
                  className="flex-1"
                />
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="shrink-0 self-start"
                  onClick={() => setEmailDialogOpen(true)}
                >
                  Change
                </Button>
              </div>
              <p className="text-xs text-muted-foreground">
                Managed through your authentication provider.
              </p>
            </div>
          </div>

          <hr className="border-border" />

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="settings-contact-email">
                Contact email
              </Label>
              <Input
                id="settings-contact-email"
                type="email"
                placeholder="you@example.com"
                value={email}
                onChange={(e) => {
                  setEmail(e.target.value);
                  setIsDirty(true);
                }}
              />
              <p className="text-xs text-muted-foreground">
                Used for email notifications about approval requests.
              </p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="settings-phone">Phone number</Label>
              <Input
                id="settings-phone"
                type="tel"
                placeholder="+15551234567"
                value={phone}
                onChange={(e) => {
                  setPhone(e.target.value);
                  setIsDirty(true);
                }}
              />
              <p className="text-xs text-muted-foreground">
                E.164 format. Used for SMS notifications.
              </p>
            </div>
          </div>

          <div className="flex justify-end">
            <Button type="submit" disabled={isLoading || !isDirty}>
              {isLoading && <Loader2 className="animate-spin" />}
              Save Changes
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>

    <EmailChangeDialog
      open={emailDialogOpen}
      onOpenChange={setEmailDialogOpen}
    />
    </>
  );
}
