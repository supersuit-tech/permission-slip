import { freePlan, paidPlan, formatLimit, PRICE_PER_REQUEST } from "@/config/plans";

/** Number of free requests included per month for all plans. */
export const FREE_REQUEST_ALLOWANCE: number = freePlan.max_requests_per_month ?? 1000;

// Re-export for backward compatibility — canonical definition is in @/config/plans.
export { PRICE_PER_REQUEST };

/** Features included in the paid (Pay-as-you-go) plan. */
export const PAID_PLAN_FEATURES = [
  "Unlimited agents",
  "Unlimited credentials",
  "Unlimited standing approvals",
  `${paidPlan.audit_retention_days}-day audit retention`,
] as const;

/** Pricing description for the paid plan. */
export const PAID_PLAN_PRICING =
  `First ${formatLimit(freePlan.max_requests_per_month)} requests/month are free. After that, $${(paidPlan.price_per_request_millicents / 100_000).toFixed(3)}/request.`;

/**
 * Free plan limits used for downgrade warnings.
 * managePath points to where users can deactivate or remove excess resources.
 */
export const FREE_PLAN_LIMITS = {
  agents: { label: "agents", limit: freePlan.max_agents, managePath: "/" },
  standing_approvals: { label: "standing approvals", limit: freePlan.max_standing_approvals, managePath: "/" },
  credentials: { label: "credentials", limit: freePlan.max_credentials, managePath: "/settings" },
} as const;
