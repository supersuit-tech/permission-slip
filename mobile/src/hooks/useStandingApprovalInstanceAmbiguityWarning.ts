import { useMemo } from "react";
import { connectorIdFromActionType } from "../screens/approvals/approvalUtils";
import {
  shouldShowStandingApprovalInstanceAmbiguityWarning,
  STANDING_APPROVAL_INSTANCE_AMBIGUITY_WARNING,
} from "../screens/approvals/standingApprovalInstanceAmbiguity";
import { useAgentConnectorInstances } from "./useAgentConnectorInstances";

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
