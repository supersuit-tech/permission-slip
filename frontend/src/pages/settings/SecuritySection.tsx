import { useCallback, useEffect, useRef, useState } from "react";
import type { Factor } from "@supabase/supabase-js";
import { Loader2, Shield } from "lucide-react";
import { useAuth } from "@/auth/AuthContext";
import { MfaEnrollmentFlow } from "@/pages/security/MfaEnrollmentFlow";
import { InlineConfirmButton } from "@/components/InlineConfirmButton";
import { Button } from "@/components/ui/button";
import { FormError } from "@/components/FormError";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

export function SecuritySection() {
  const { listMfaFactors, unenrollMfa } = useAuth();
  const [factors, setFactors] = useState<Factor[]>([]);
  const [view, setView] = useState<"loading" | "error" | "enrolled" | "enroll">(
    "loading",
  );
  const [isRemoving, setIsRemoving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const hasLoadedRef = useRef(false);
  const mountedRef = useRef(true);

  useEffect(() => {
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const loadFactors = useCallback(async () => {
    setView("loading");
    setError(null);
    const { factors: result, error: loadError } = await listMfaFactors();
    if (!mountedRef.current) return;
    if (loadError) {
      setError("Failed to load security settings.");
      setView("error");
    } else {
      const verified = result.filter((f) => f.status === "verified");
      setFactors(verified);
      setView(verified.length > 0 ? "enrolled" : "enroll");
    }
    hasLoadedRef.current = true;
  }, [listMfaFactors]);

  useEffect(() => {
    if (!hasLoadedRef.current) {
      loadFactors();
    }
  }, [loadFactors]);

  const handleEnrolled = useCallback(async () => {
    const { factors: result, error: refreshError } = await listMfaFactors();
    if (!mountedRef.current) return;
    if (!refreshError) {
      const verified = result.filter((f) => f.status === "verified");
      setFactors(verified);
      setView(verified.length > 0 ? "enrolled" : "enroll");
    } else {
      setView("enrolled");
    }
  }, [listMfaFactors]);

  const handleRemove = async (factorId: string) => {
    setIsRemoving(true);
    setError(null);
    const { error: removeError } = await unenrollMfa(factorId);
    if (!mountedRef.current) return;
    if (removeError) {
      setError("Failed to remove authenticator. Please try again.");
    } else {
      hasLoadedRef.current = false;
      await loadFactors();
    }
    if (mountedRef.current) {
      setIsRemoving(false);
    }
  };

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
        <div className="space-y-4">
          <h3 className="text-sm font-medium">Two-Factor Authentication</h3>
          {view === "loading" ? (
            <div
              className="flex items-center justify-center py-8"
              role="status"
              aria-label="Loading security settings"
            >
              <Loader2 className="text-muted-foreground size-5 animate-spin" />
            </div>
          ) : view === "error" ? (
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
          ) : view === "enrolled" ? (
            <div className="space-y-3">
              {factors.map((factor) => (
                <div
                  key={factor.id}
                  className="flex items-center justify-between rounded-lg border p-4"
                >
                  <div className="space-y-0.5">
                    <p className="text-sm font-medium">
                      {factor.friendly_name ?? "Authenticator App"}
                    </p>
                    <p className="text-xs text-green-600 dark:text-green-400">
                      Enabled
                    </p>
                  </div>
                  <InlineConfirmButton
                    confirmLabel="Confirm"
                    isProcessing={isRemoving}
                    onConfirm={() => handleRemove(factor.id)}
                  >
                    <Button variant="outline" size="sm">
                      Remove
                    </Button>
                  </InlineConfirmButton>
                </div>
              ))}
            </div>
          ) : (
            <MfaEnrollmentFlow onEnrolled={handleEnrolled} />
          )}
        </div>
      </CardContent>
    </Card>
  );
}
