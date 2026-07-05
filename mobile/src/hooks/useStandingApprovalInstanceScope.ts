import { useMemo } from "react";
import { connectorIdFromActionType } from "../screens/approvals/approvalUtils";
import { getStandingApprovalInstanceScopeLabel } from "../screens/approvals/standingApprovalInstanceAmbiguity";
import { useAgentConnectorInstances } from "./useAgentConnectorInstances";

export function useStandingApprovalInstanceScope(request: {
  agent_id: number;
  action_type: string;
  connector_instance_display?: string | null;
}): { scopeLabel: string | null } {
  const connectorId = connectorIdFromActionType(request.action_type);
  const { instances, isLoading } = useAgentConnectorInstances(
    request.agent_id,
    connectorId,
  );

  const scopeLabel = useMemo(
    () =>
      getStandingApprovalInstanceScopeLabel(
        request.connector_instance_display,
        instances,
        !isLoading,
      ),
    [request.connector_instance_display, instances, isLoading],
  );

  return { scopeLabel };
}
