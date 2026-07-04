import { useMemo } from "react";
import { useAgentConnectorInstances } from "@/hooks/useAgentConnectorInstances";
import {
  shouldShowStandingApprovalInstanceAmbiguityWarning,
  STANDING_APPROVAL_INSTANCE_AMBIGUITY_WARNING,
} from "@/pages/dashboard/standingApprovalInstanceAmbiguity";

function connectorIdFromActionType(actionType: string): string {
  const dotIndex = actionType.indexOf(".");
  return dotIndex > 0 ? actionType.substring(0, dotIndex) : actionType;
}

export function useStandingApprovalInstanceAmbiguityWarning(request: {
  agent_id: number;
  action_type: string;
  connector_instance_display?: string | null;
}): { showWarning: boolean; warningMessage: string } {
  const connectorId = connectorIdFromActionType(request.action_type);
  const { instances, isLoading } = useAgentConnectorInstances(
    request.agent_id,
    connectorId,
  );

  const showWarning = useMemo(
    () =>
      shouldShowStandingApprovalInstanceAmbiguityWarning(
        request.connector_instance_display,
        isLoading ? undefined : instances.length,
      ),
    [request.connector_instance_display, instances.length, isLoading],
  );

  return {
    showWarning,
    warningMessage: STANDING_APPROVAL_INSTANCE_AMBIGUITY_WARNING,
  };
}
