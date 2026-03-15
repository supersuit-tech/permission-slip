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

/** Free plan has finite limits for all resource fields. */
interface FreePlanConfig extends PlanConfig {
  max_requests_per_month: number;
  max_agents: number;
  max_standing_approvals: number;
  max_credentials: number;
}

type PlansMap = Record<string, PlanConfig>;

const plans: PlansMap = plansJson;

export type { PlanConfig, FreePlanConfig };

const freePlanRaw = plans["free"];
const paidPlanRaw = plans["pay_as_you_go"];
if (!freePlanRaw || !paidPlanRaw) {
  throw new Error("plans.json is missing required plan definitions (free, pay_as_you_go)");
}
if (
  freePlanRaw.max_requests_per_month == null ||
  freePlanRaw.max_agents == null ||
  freePlanRaw.max_standing_approvals == null ||
  freePlanRaw.max_credentials == null
) {
  throw new Error(
    "plans.json free plan is missing required limit fields (max_requests_per_month, max_agents, max_standing_approvals, max_credentials)"
  );
}
export const freePlan: FreePlanConfig = freePlanRaw as FreePlanConfig;
export const paidPlan: PlanConfig = paidPlanRaw;

/** Format a number with locale-aware separators (e.g. 1,000). */
export function formatLimit(n: number): string {
  return n.toLocaleString();
}

/** Formatted per-request price string (e.g. "$0.005"), derived from plans.json. */
export const PRICE_PER_REQUEST = `$${(paidPlan.price_per_request_millicents / 100_000).toFixed(3)}`;
