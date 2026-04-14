import type { StandingApproval } from "@/hooks/useStandingApprovals";

export type StandingApprovalRowStatus =
  | "none"
  | "active"
  | "expired"
  | "revoked";

/**
 * Pick the most relevant standing approval row for display (prefer active).
 */
export function pickPrimaryStandingApproval(
  rows: StandingApproval[],
): StandingApproval | null {
  if (rows.length === 0) return null;
  const active = rows.find((r) => r.status === "active");
  if (active) return active;
  return rows[0] ?? null;
}

export function standingApprovalRowStatus(
  rows: StandingApproval[],
): StandingApprovalRowStatus {
  const primary = pickPrimaryStandingApproval(rows);
  if (!primary) return "none";
  switch (primary.status) {
    case "active":
      return "active";
    case "expired":
      return "expired";
    case "revoked":
      return "revoked";
    default:
      return "none";
  }
}

export function standingApprovalStatusLabel(
  status: StandingApprovalRowStatus,
): string {
  switch (status) {
    case "active":
      return "Active";
    case "none":
      return "None";
    case "expired":
      return "Expired";
    case "revoked":
      return "Revoked";
    default:
      return "None";
  }
}
