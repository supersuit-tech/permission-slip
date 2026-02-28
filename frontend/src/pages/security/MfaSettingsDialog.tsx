import { useCallback, useEffect, useRef, useState } from "react";
import type { Factor } from "@supabase/supabase-js";
import { useAuth } from "@/auth/AuthContext";
import { MfaEnrollmentFlow } from "./MfaEnrollmentFlow";
import { Button } from "@/components/ui/button";
import { FormError } from "@/components/FormError";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

type View = "loading" | "error" | "enrolled" | "enroll" | "just_enrolled";

// ---------------------------------------------------------------------------
// Sub-components for each dialog view state
// ---------------------------------------------------------------------------

/** Shown immediately after the user finishes MFA enrollment. */
function EnrollmentSuccessView({
  onDone,
}: {
  onDone: () => void;
}) {
  return (
    <div className="space-y-3 text-center">
      <p className="text-sm font-medium text-green-600 dark:text-green-400">
        Authenticator app enrolled successfully.
      </p>
      <p className="text-sm text-muted-foreground">
        You&apos;ll be asked for a code from your authenticator app each time
        you sign in.
      </p>
      <Button variant="outline" size="sm" onClick={onDone}>
        Done
      </Button>
    </div>
  );
}

/** Lists verified MFA factors with a remove button for each. */
function FactorList({
  factors,
  isRemoving,
  onRemove,
}: {
  factors: Factor[];
  isRemoving: boolean;
  onRemove: (factorId: string) => void;
}) {
  return (
    <div className="space-y-3">
      {factors.map((factor) => (
        <div
          key={factor.id}
          className="flex items-center justify-between rounded-md border p-3"
        >
          <div>
            <p className="text-sm font-medium">
              {factor.friendly_name ?? "Authenticator App"}
            </p>
            <p className="text-xs text-green-600 dark:text-green-400">
              Enabled
            </p>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={() => onRemove(factor.id)}
            disabled={isRemoving}
          >
            Remove
          </Button>
        </div>
      ))}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main dialog
// ---------------------------------------------------------------------------

interface MfaSettingsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function MfaSettingsDialog({
  open,
  onOpenChange,
}: MfaSettingsDialogProps) {
  const { listMfaFactors, unenrollMfa } = useAuth();
  const [factors, setFactors] = useState<Factor[]>([]);
  const [view, setView] = useState<View>("loading");
  const [isRemoving, setIsRemoving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Track whether we've already loaded once for this dialog open.
  // This prevents re-fetching when auth state changes cause re-renders
  // (e.g. after MFA enrollment updates the session, which recreates
  // listMfaFactors → loadFactors → useEffect fires again).
  const hasLoadedRef = useRef(false);

  // Mirror `open` into a ref so async callbacks can check whether the
  // dialog is still open without depending on the prop directly (which
  // would break useCallback memoization).
  const openRef = useRef(open);
  openRef.current = open;

  const loadFactors = useCallback(async () => {
    setView("loading");
    setError(null);
    const { factors, error } = await listMfaFactors();
    // If the dialog was closed while the request was in-flight, bail out
    // to avoid repopulating state after handleOpenChange has reset it.
    if (!openRef.current) return;
    if (error) {
      setError("Failed to load security settings.");
      setView("error");
    } else {
      const verified = factors.filter((f) => f.status === "verified");
      setFactors(verified);
      setView(verified.length > 0 ? "enrolled" : "enroll");
    }
    hasLoadedRef.current = true;
  }, [listMfaFactors]);

  useEffect(() => {
    if (open && !hasLoadedRef.current) {
      loadFactors();
    }
    if (!open) {
      // Reset when dialog closes so next open fetches fresh data.
      hasLoadedRef.current = false;
    }
  }, [open, loadFactors]);

  const handleEnrolled = useCallback(async () => {
    // Show success state immediately — no loading flash.
    setView("just_enrolled");
    hasLoadedRef.current = true; // prevent useEffect from overriding
    // Refresh factors in the background so the list is ready when the
    // user dismisses the success message. On failure, keep the success
    // view — the factors will be fetched when the user clicks "Done".
    const { factors, error: refreshError } = await listMfaFactors();
    // Skip the state update if the dialog was closed while we were waiting.
    if (!refreshError && openRef.current) {
      setFactors(factors.filter((f) => f.status === "verified"));
    }
  }, [listMfaFactors]);

  const handleRemove = async (factorId: string) => {
    setIsRemoving(true);
    setError(null);
    const { error } = await unenrollMfa(factorId);

    // If the dialog was closed while the request was in-flight, bail out
    // to avoid state updates on the stale view.
    if (!openRef.current) return;

    if (error) {
      setError("Failed to remove authenticator. Please try again.");
    } else {
      hasLoadedRef.current = false; // allow loadFactors to refresh
      await loadFactors();
    }

    if (openRef.current) {
      setIsRemoving(false);
    }
  };

  const handleDoneAfterEnroll = useCallback(() => {
    if (factors.length === 0) {
      hasLoadedRef.current = false;
      loadFactors();
    } else {
      setView("enrolled");
    }
  }, [factors.length, loadFactors]);

  // Prevent Radix from closing the dialog on pointer-down-outside or
  // focus-outside (e.g. switching apps on mobile, clicking another window).
  // The user must explicitly click Close or press Escape.
  const handleInteractOutside = (e: Event) => {
    e.preventDefault();
  };

  const handleOpenChange = useCallback(
    (nextOpen: boolean) => {
      if (!nextOpen) {
        setFactors([]);
        setView("loading");
        setIsRemoving(false);
        setError(null);
        hasLoadedRef.current = false;
      }
      onOpenChange(nextOpen);
    },
    [onOpenChange]
  );

  const renderContent = () => {
    switch (view) {
      case "loading":
        return <p className="text-sm text-muted-foreground">Loading...</p>;
      case "error":
        return (
          <div className="space-y-3">
            <FormError error={error} />
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                hasLoadedRef.current = false;
                loadFactors();
              }}
            >
              Try Again
            </Button>
          </div>
        );
      case "just_enrolled":
        return <EnrollmentSuccessView onDone={handleDoneAfterEnroll} />;
      case "enrolled":
        return (
          <FactorList
            factors={factors}
            isRemoving={isRemoving}
            onRemove={handleRemove}
          />
        );
      case "enroll":
        return <MfaEnrollmentFlow onEnrolled={handleEnrolled} />;
      default: {
        const _exhaustive: never = view;
        return _exhaustive;
      }
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent onInteractOutside={handleInteractOutside}>
        <DialogHeader>
          <DialogTitle>Security Settings</DialogTitle>
          <DialogDescription>
            Manage your account security and authentication methods.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <h3 className="text-sm font-medium">Two-Factor Authentication</h3>
          {renderContent()}
        </div>
      </DialogContent>
    </Dialog>
  );
}
