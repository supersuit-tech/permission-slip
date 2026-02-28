import { useMemo } from "react";
import { useAgents, type Agent } from "./useAgents";

/**
 * Returns only pending agents by filtering the shared agent list from useAgents.
 *
 * This composes useAgents rather than issuing its own query, so the fetch logic
 * and query key live in a single place. Both hooks share the ["agents"] cache
 * and its 5-second polling interval while pending agents exist.
 */
export function usePendingAgents(): {
  pendingAgents: Agent[];
  isLoading: boolean;
} {
  const { agents, isLoading } = useAgents();

  const pendingAgents = useMemo(
    () =>
      agents.filter((a) => {
        if (a.status !== "pending") return false;
        if (a.expires_at && new Date(a.expires_at).getTime() <= Date.now())
          return false;
        return true;
      }),
    [agents],
  );

  return { pendingAgents, isLoading };
}
