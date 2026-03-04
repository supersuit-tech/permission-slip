/** Features included in the paid (Pay-as-you-go) plan. */
export const PAID_PLAN_FEATURES = [
  "Unlimited agents",
  "Unlimited credentials",
  "Unlimited standing approvals",
  "90-day audit retention",
] as const;

/** Pricing description for the paid plan. */
export const PAID_PLAN_PRICING =
  "First 1,000 requests/month are free. After that, $0.005 per request.";

/**
 * Free plan limits used for downgrade warnings.
 * managePath points to where users can deactivate or remove excess resources.
 */
export const FREE_PLAN_LIMITS = {
  agents: { label: "agents", limit: 3, managePath: "/" },
  standing_approvals: { label: "standing approvals", limit: 5, managePath: "/" },
  credentials: { label: "credentials", limit: 5, managePath: "/settings" },
} as const;
