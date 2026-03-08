import { useState } from "react";
import { Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { CredentialSummary } from "@/hooks/useCredentials";
import { RemoveCredentialDialog } from "./RemoveCredentialDialog";

interface StoredCredentialListProps {
  credentials: CredentialSummary[];
  /** Fallback label when credential.label is null (default: credential.service). */
  defaultLabel?: string;
}

export function StoredCredentialList({
  credentials,
  defaultLabel,
}: StoredCredentialListProps) {
  const [removeTarget, setRemoveTarget] = useState<CredentialSummary | null>(
    null,
  );

  if (credentials.length === 0) return null;

  return (
    <>
      <div className="space-y-2">
        {credentials.map((cred) => {
          const label = cred.label ?? defaultLabel ?? cred.service;
          return (
            <div
              key={cred.id}
              className="bg-muted/50 flex items-center justify-between rounded-md px-3 py-2"
            >
              <div className="min-w-0">
                <p className="truncate text-sm">{label}</p>
                <p className="text-muted-foreground text-xs">
                  Added {new Date(cred.created_at).toLocaleDateString()}
                </p>
              </div>
              <Button
                variant="ghost"
                size="sm"
                className="text-destructive hover:text-destructive"
                onClick={() => setRemoveTarget(cred)}
                aria-label={`Remove credential ${label}`}
              >
                <Trash2 className="size-4" />
              </Button>
            </div>
          );
        })}
      </div>

      {removeTarget && (
        <RemoveCredentialDialog
          open={!!removeTarget}
          onOpenChange={(open) => {
            if (!open) setRemoveTarget(null);
          }}
          credential={removeTarget}
        />
      )}
    </>
  );
}
