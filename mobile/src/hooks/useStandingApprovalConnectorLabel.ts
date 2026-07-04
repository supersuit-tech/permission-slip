import { useMemo } from "react";
import { useConnectorDisplayName } from "./useConnectorDisplayName";
import { formatConnectorDisplayName } from "../screens/approvals/approvalUtils";
import type { components } from "../api/schema";

type StandingApprovalRequest = components["schemas"]["StandingApprovalRequest"];

/**
 * Resolves the connector label for a standing approval request, preferring
 * server-frozen display fields when present and falling back to client lookup.
 */
export function useStandingApprovalConnectorLabel(
  request: Pick<
    StandingApprovalRequest,
    "action_type" | "connector_name" | "connector_instance_display"
  >,
): { connectorLabel: string } {
  const { connectorDisplayName } = useConnectorDisplayName(request.action_type);

  const connectorLabel = useMemo(
    () =>
      formatConnectorDisplayName({
        connectorName: request.connector_name ?? connectorDisplayName,
        actionType: request.action_type,
        instanceDisplay: request.connector_instance_display,
      }),
    [
      request.connector_name,
      request.connector_instance_display,
      request.action_type,
      connectorDisplayName,
    ],
  );

  return { connectorLabel };
}
