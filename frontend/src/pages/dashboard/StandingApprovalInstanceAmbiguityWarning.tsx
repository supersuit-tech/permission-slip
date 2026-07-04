import { AlertTriangle } from "lucide-react";
import { STANDING_APPROVAL_INSTANCE_AMBIGUITY_WARNING } from "./standingApprovalInstanceAmbiguity";

interface StandingApprovalInstanceAmbiguityWarningProps {
  className?: string;
}

export function StandingApprovalInstanceAmbiguityWarning({
  className,
}: StandingApprovalInstanceAmbiguityWarningProps) {
  return (
    <div
      role="alert"
      className={
        className ??
        "flex items-start gap-2 rounded-md border border-amber-500/30 bg-amber-500/5 px-3 py-2 text-sm text-amber-800 dark:text-amber-300"
      }
    >
      <AlertTriangle className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
      <span>{STANDING_APPROVAL_INSTANCE_AMBIGUITY_WARNING}</span>
    </div>
  );
}
