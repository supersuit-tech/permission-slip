import { useBillingPlan } from "./useBillingPlan";

type PlanLimitKey = "max_agents" | "max_standing_approvals" | "max_credentials";
type UsageKey = "agents" | "standing_approvals" | "credentials";

const LIMIT_TO_USAGE: Record<PlanLimitKey, UsageKey> = {
  max_agents: "agents",
  max_standing_approvals: "standing_approvals",
  max_credentials: "credentials",
};

interface ResourceLimitResult {
  /** The plan limit, or null if unlimited. */
  max: number | null;
  /** Current usage count from billing data, or the fallback count. */
  current: number;
  /** Whether the user has reached or exceeded the limit. */
  atLimit: boolean;
  /** Whether billing plan data has loaded (controls badge visibility). */
  hasData: boolean;
}

/**
 * Extracts a specific resource limit and current usage from the billing plan.
 *
 * @param limitKey - The plan limit field (e.g. "max_agents")
 * @param fallbackCount - Used when billing data isn't available yet
 */
export function useResourceLimit(
  limitKey: PlanLimitKey,
  fallbackCount: number,
): ResourceLimitResult {
  const { billingPlan } = useBillingPlan();

  const usageKey = LIMIT_TO_USAGE[limitKey];
  const max =
    billingPlan?.effective_limits?.[limitKey] ?? billingPlan?.plan?.[limitKey] ?? null;
  const current = billingPlan?.usage?.[usageKey] ?? fallbackCount;
  const atLimit = max != null && current >= max;
  const hasData = billingPlan?.plan != null && billingPlan?.effective_limits != null;

  return { max, current, atLimit, hasData };
}
