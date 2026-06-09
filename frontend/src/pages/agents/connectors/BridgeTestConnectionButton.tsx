import { useState } from "react";
import { Loader2, PlugZap } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { useTestCredentialConnection } from "@/hooks/useTestCredentialConnection";

interface BridgeTestConnectionButtonProps {
  service: string;
  credentialId?: string;
  buildCredentials: () => Record<string, string> | null;
  disabled?: boolean;
}

export function BridgeTestConnectionButton({
  service,
  credentialId,
  buildCredentials,
  disabled,
}: BridgeTestConnectionButtonProps) {
  const { testConnection, isTesting } = useTestCredentialConnection();
  const [lastResult, setLastResult] = useState<"ok" | "error" | null>(null);

  if (service !== "protonmail") {
    return null;
  }

  async function handleTest() {
    const credentials = buildCredentials();
    if (!credentials && !credentialId) {
      toast.error("Enter Bridge username and password to test");
      return;
    }

    try {
      const result = await testConnection({
        service: "protonmail",
        credential_id: credentialId,
        credentials: credentials ?? undefined,
      });
      setLastResult("ok");
      toast.success(result?.message ?? "Bridge connection successful");
    } catch (err) {
      setLastResult("error");
      toast.error(
        err instanceof Error ? err.message : "Bridge connection test failed",
      );
    }
  }

  return (
    <div className="space-y-2">
      <Button
        type="button"
        variant="outline"
        onClick={handleTest}
        disabled={disabled || isTesting}
        className="w-full"
      >
        {isTesting ? (
          <Loader2 className="animate-spin" />
        ) : (
          <PlugZap className="size-4" />
        )}
        Test Bridge connection
      </Button>
      {lastResult === "ok" ? (
        <p className="text-xs text-green-600 dark:text-green-400">
          Bridge is reachable over IMAP and SMTP.
        </p>
      ) : null}
      {lastResult === "error" ? (
        <p className="text-muted-foreground text-xs">
          Fix the values above and try again. Run{" "}
          <code className="text-xs">protonmail-bridge info</code> for the correct
          host, port, username, and Bridge password.
        </p>
      ) : null}
    </div>
  );
}
