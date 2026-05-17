import { Shield } from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

export function SecuritySection() {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <Shield className="text-muted-foreground size-5" />
          <CardTitle>Security</CardTitle>
        </div>
        <CardDescription>
          Manage your account security and authentication methods.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <p className="text-sm text-muted-foreground">
          Two-factor authentication and authenticator enrollment are not
          available in this deployment. Your account is protected by your
          password and session tokens stored on this device.
        </p>
      </CardContent>
    </Card>
  );
}
