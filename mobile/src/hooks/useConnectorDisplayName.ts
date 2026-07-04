import { useMemo } from "react";
import { useConnectorDetail } from "./useConnectorDetail";
import {
  connectorIdFromActionType,
  formatConnectorDisplayName,
} from "../screens/approvals/approvalUtils";

/**
 * Resolves a human-readable connector name for an action type, using the
 * connector manifest name when available and falling back to the action prefix.
 */
export function useConnectorDisplayName(actionType: string): {
  connectorDisplayName: string;
  isLoading: boolean;
} {
  const connectorId = connectorIdFromActionType(actionType);
  const { connector, isLoading } = useConnectorDetail(connectorId);

  const connectorDisplayName = useMemo(
    () =>
      formatConnectorDisplayName({
        connectorName: connector?.name,
        actionType,
      }),
    [connector?.name, actionType],
  );

  return { connectorDisplayName, isLoading };
}
