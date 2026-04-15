import { useAgents } from "./useAgents";
import { useAgentConnectors } from "./useAgentConnectors";

/**
 * Returns whether the current user has at least one registered agent
 * with at least one connector configured. Used to gate the iOS app
 * download link in the settings navigation.
 */
export function useIsFullyConfigured(): {
  isFullyConfigured: boolean;
  isLoading: boolean;
} {
  const { agents, isLoading: agentsLoading } = useAgents();

  const registeredAgents = agents.filter((a) => a.status === "registered");
  const firstAgentId = registeredAgents[0]?.agent_id ?? 0;

  const { connectors, isLoading: connectorsLoading } =
    useAgentConnectors(firstAgentId);

  const isLoading = agentsLoading || (firstAgentId > 0 && connectorsLoading);

  const isFullyConfigured =
    !isLoading && firstAgentId > 0 && connectors.length > 0;

  return { isFullyConfigured, isLoading };
}
