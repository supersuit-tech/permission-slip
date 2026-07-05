import { useMemo } from "react";
import { useAgentConnectorInstances } from "@/hooks/useAgentConnectorInstances";
import { getStandingApprovalInstanceScopeLabel } from "@/pages/dashboard/standingApprovalInstanceAmbiguity";

function connectorIdFromActionType(actionType: string): string {
  const dotIndex = actionType.indexOf(".");
  return dotIndex > 0 ? actionType.substring(0, dotIndex) : actionType;
}

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
