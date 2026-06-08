import { Shield, ShieldAlert, ShieldCheck } from "lucide-react";
import { Badge } from "@/components/ui/badge";

export type RiskLevel = "low" | "medium" | "high";

export function RiskBadge({ level }: { level?: RiskLevel }) {
  switch (level) {
    case "low":
      return (
        <Badge
          variant="secondary"
          className="bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
        >
          <ShieldCheck className="size-3" />
          Low
        </Badge>
      );
    case "medium":
      return (
        <Badge
          variant="secondary"
          className="bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400"
        >
          <Shield className="size-3" />
          Medium
        </Badge>
      );
    case "high":
      return (
        <Badge variant="destructive">
          <ShieldAlert className="size-3" />
          High
        </Badge>
      );
    default:
      return (
        <Badge variant="outline">
          <Shield className="size-3" />
          Unknown
        </Badge>
      );
  }
}

/** One-line, risk-aware explanation of what selecting this action means. */
export function riskBlurb(level?: RiskLevel): string {
  switch (level) {
    case "high":
      return "High-risk: this action can make irreversible or externally-visible changes (e.g. sending email). Review carefully.";
    case "medium":
      return "Medium-risk: this action modifies state and may be hard to undo.";
    case "low":
      return "Low-risk: this is a read-only or low-impact action.";
    default:
      return "Risk level for this action is unknown.";
  }
}

/** Accent border for the whole dialog so the risk tier is ambient while editing. */
export function riskDialogAccentClass(level?: RiskLevel): string {
  switch (level) {
    case "high":
      return "border-t-4 border-t-destructive";
    case "medium":
      return "border-t-4 border-t-yellow-500";
    case "low":
      return "border-t-4 border-t-green-500";
    default:
      return "";
  }
}

/** Background/border tint for the inline risk info card. */
export function riskCardClass(level?: RiskLevel): string {
  switch (level) {
    case "high":
      return "bg-destructive/10 border-destructive/20";
    case "medium":
      return "bg-yellow-100/50 border-yellow-300 dark:bg-yellow-900/20 dark:border-yellow-900/40";
    case "low":
      return "bg-green-100/40 border-green-300 dark:bg-green-900/20 dark:border-green-900/40";
    default:
      return "bg-muted/50";
  }
}
