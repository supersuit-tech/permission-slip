import { useMemo } from "react";
import { useConnectorDetail } from "./useConnectorDetail";

function connectorIdFromActionType(actionType: string): string {
  const dotIndex = actionType.indexOf(".");
  return dotIndex > 0 ? actionType.substring(0, dotIndex) : actionType;
}

/**
 * Fetches display metadata for a given action type by looking up the
 * connector detail and finding the matching action.
 */
export function useActionSchema(actionType: string): {
  displayTemplate: string | null;
  actionName: string | null;
  isLoading: boolean;
} {
  const connectorId = connectorIdFromActionType(actionType);
  const { connector, isLoading } = useConnectorDetail(connectorId);

  const result = useMemo(() => {
    if (!connector?.actions) {
      return { displayTemplate: null, actionName: null };
    }

    const action = connector.actions.find((a) => a.action_type === actionType);
    if (!action) {
      return { displayTemplate: null, actionName: null };
    }

    return {
      displayTemplate: action.display_template ?? null,
      actionName: action.name ?? null,
    };
  }, [connector, actionType]);

  return {
    ...result,
    isLoading,
  };
}
