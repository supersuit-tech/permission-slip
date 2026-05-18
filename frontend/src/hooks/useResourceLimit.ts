export type PlanLimitKey = "max_agents" | "max_standing_approvals" | "max_credentials";

type UsageKey = "agents" | "standing_approvals" | "credentials";

const LIMIT_TO_USAGE: Record<PlanLimitKey, UsageKey> = {
  max_agents: "agents",
  max_standing_approvals: "standing_approvals",
  max_credentials: "credentials",
};

interface ResourceLimitResult {
  /** The plan limit, or null if unlimited. */
  max: number | null;
  /** Current usage count. */
  current: number;
  /** Whether the user has reached or exceeded the limit. */
  atLimit: boolean;
  /** Whether limit data is available for UI badges. */
  hasData: boolean;
}

/**
 * Self-hosted Permission Slip uses unlimited plans — no billing API.
 * Returns unlimited limits while still surfacing the current count from callers.
 */
export function useResourceLimit(
  limitKey: PlanLimitKey,
  fallbackCount: number,
): ResourceLimitResult {
  void LIMIT_TO_USAGE[limitKey];
  return {
    max: null,
    current: fallbackCount,
    atLimit: false,
    hasData: true,
  };
}
