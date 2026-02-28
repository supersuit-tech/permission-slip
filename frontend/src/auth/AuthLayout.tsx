import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

interface AuthLayoutProps {
  children: ReactNode;
}

export default function AuthLayout({ children }: AuthLayoutProps) {
  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <div className="w-full max-w-sm space-y-4">
        <Card>
          <CardHeader className="items-center text-center">
            <span className="mb-1 flex h-10 w-10 items-center justify-center rounded-lg bg-primary text-lg font-bold text-primary-foreground">
              P
            </span>
            <CardTitle className="text-2xl">Permission Slip</CardTitle>
            <CardDescription>
              Sign in to manage your permissions
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">{children}</CardContent>
        </Card>
        <div className="flex justify-center gap-4 text-xs text-muted-foreground">
          <Link to="/policy/privacy" className="hover:text-foreground transition-colors">Privacy</Link>
          <Link to="/policy/terms" className="hover:text-foreground transition-colors">Terms</Link>
          <Link to="/policy/cookies" className="hover:text-foreground transition-colors">Cookies</Link>
        </div>
      </div>
    </div>
  );
}
