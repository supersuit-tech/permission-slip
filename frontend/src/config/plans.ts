/**
 * Frontend access to the shared plan configuration.
 *
 * Imports config/plans.json (repo root) via the @config alias so the
 * frontend stays in sync with the Go backend automatically.
 */
import plansJson from "@config/plans.json";

interface PlanConfig {
  name: string;
  max_requests_per_month: number | null;
  max_agents: number | null;
  max_standing_approvals: number | null;
  max_credentials: number | null;
  audit_retention_days: number;
  price_per_request_millicents: number;
}

type PlansMap = Record<string, PlanConfig>;

const plans: PlansMap = plansJson;

export type { PlanConfig };

export const freePlan = plans["free"]!;
export const paidPlan = plans["pay_as_you_go"]!;

/** Format a number with locale-aware separators (e.g. 1,000). */
export function formatLimit(n: number): string {
  return n.toLocaleString();
}
