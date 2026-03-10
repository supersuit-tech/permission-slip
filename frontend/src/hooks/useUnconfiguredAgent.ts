import type { Agent } from "./useAgents";
import { useAgentConnectors } from "./useAgentConnectors";
import { getAgentDisplayName } from "@/lib/agents";

/**
 * Detects the "unconfigured agent" state: exactly one registered agent
 * with zero connectors enabled. Used by the dashboard to show a focused
 * PLG experience guiding the user to configure their agent.
 */
export function useUnconfiguredAgent(agents: Agent[], agentsLoading: boolean) {
  const registeredAgents = agents.filter((a) => a.status === "registered");
  const singleRegistered =
    !agentsLoading && registeredAgents.length === 1
      ? registeredAgents[0]
      : null;

  const agentId = singleRegistered?.agent_id ?? 0;

  // useAgentConnectors guards with `enabled: agentId > 0`, so passing 0
  // prevents any network request when the criteria aren't met.
  const {
    connectors,
    isLoading: connectorsLoading,
    error: connectorsError,
  } = useAgentConnectors(agentId);

  // Only report loading during the connector fetch phase (not during agents
  // loading). The Dashboard already handles the agents loading state via its
  // own isLoading. Including agentsLoading here would suppress dashboard cards
  // for ALL users during the initial load, not just the single-agent case.
  const isLoading = agentId > 0 && connectorsLoading;
  const isUnconfigured =
    !isLoading && agentId > 0 && connectors.length === 0 && !connectorsError;

  return {
    isUnconfigured,
    isLoading,
    agentId,
    agentName: singleRegistered
      ? getAgentDisplayName(singleRegistered)
      : undefined,
  };
}
