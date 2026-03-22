import type { UsageSummary } from "@/hooks/useBillingPlan";
import { FREE_PLAN_LIMITS } from "./constants";

export interface LimitWarning {
  resource: string;
  current: number;
  limit: number;
  managePath: string;
}

/** Compare current usage against free plan limits and return warnings for resources over the limit. */
export function buildLimitWarnings(usage: UsageSummary | null): LimitWarning[] {
  if (!usage) return [];
  const warnings: LimitWarning[] = [];
  if (usage.agents > FREE_PLAN_LIMITS.agents.limit) {
    warnings.push({
      resource: FREE_PLAN_LIMITS.agents.label,
      current: usage.agents,
      limit: FREE_PLAN_LIMITS.agents.limit,
      managePath: FREE_PLAN_LIMITS.agents.managePath,
    });
  }
  if (usage.standing_approvals > FREE_PLAN_LIMITS.standing_approvals.limit) {
    warnings.push({
      resource: FREE_PLAN_LIMITS.standing_approvals.label,
      current: usage.standing_approvals,
      limit: FREE_PLAN_LIMITS.standing_approvals.limit,
      managePath: FREE_PLAN_LIMITS.standing_approvals.managePath,
    });
  }
  if (usage.credentials > FREE_PLAN_LIMITS.credentials.limit) {
    warnings.push({
      resource: FREE_PLAN_LIMITS.credentials.label,
      current: usage.credentials,
      limit: FREE_PLAN_LIMITS.credentials.limit,
      managePath: FREE_PLAN_LIMITS.credentials.managePath,
    });
  }
  return warnings;
}
