import type { RiskLevel } from "@/pages/agents/connectors/RiskBadge";

/** Treat missing risk metadata as high — strongest warning, never a block. */
export function effectiveRiskLevel(level?: RiskLevel | string | null): RiskLevel {
  if (level === "low" || level === "medium" || level === "high") {
    return level;
  }
  return "high";
}

export function unrestrictedApprovalWarning(level?: RiskLevel | string | null): string {
  const risk = effectiveRiskLevel(level);
  const shared =
    "This standing approval authorizes any parameters for this action. No approval prompts will be sent.";
  switch (risk) {
    case "low":
      return `${shared} This is a low-risk action (typically read-only).`;
    case "medium":
      return `${shared} This is a medium-risk action that can modify data and may be hard to undo.`;
    case "high":
      return `${shared} This is treated as a high-risk action and can make irreversible or externally-visible changes.`;
  }
}
