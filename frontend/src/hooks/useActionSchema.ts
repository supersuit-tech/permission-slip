import { useMemo } from "react";
import { useConnectorDetail } from "@/hooks/useConnectorDetail";
import {
  parseParametersSchema,
  type ParametersSchema,
} from "@/lib/parameterSchema";

/**
 * Extracts the connector ID from an action type string.
 * e.g., "github.create_issue" → "github"
 */
function connectorIdFromActionType(actionType: string): string {
  const dotIndex = actionType.indexOf(".");
  return dotIndex > 0 ? actionType.substring(0, dotIndex) : actionType;
}

/**
 * Fetches the parameters schema for a given action type by looking up
 * the connector detail and finding the matching action.
 */
export function useActionSchema(actionType: string): {
  schema: ParametersSchema | null;
  actionName: string | null;
  isLoading: boolean;
} {
  const connectorId = connectorIdFromActionType(actionType);
  const { connector, isLoading } = useConnectorDetail(connectorId);

  const result = useMemo(() => {
    if (!connector?.actions) {
      return { schema: null, actionName: null };
    }

    const action = connector.actions.find(
      (a) => a.action_type === actionType,
    );
    if (!action) {
      return { schema: null, actionName: null };
    }

    // Safe cast: the generated API type for parameters_schema is
    // `Record<string, unknown> | undefined` (additionalProperties: true).
    const schema = parseParametersSchema(
      action.parameters_schema as Record<string, unknown> | undefined,
    );

    return {
      schema,
      actionName: action.name ?? null,
    };
  }, [connector, actionType]);

  return {
    ...result,
    isLoading,
  };
}
