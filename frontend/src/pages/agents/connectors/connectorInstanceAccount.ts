import { isPatternWrapper } from "@/lib/constraints";
import type { AgentConnectorInstance } from "@/hooks/useAgentConnectorInstances";

export const CONNECTOR_INSTANCE_PARAM = "connector_instance";

export function isAllAccountsConnectorInstance(value: unknown): boolean {
  return value === undefined || value === null || value === "*";
}

export function resolveConnectorInstanceAccountLabel(
  connectorInstance: unknown,
  instances: AgentConnectorInstance[],
): string {
  if (isAllAccountsConnectorInstance(connectorInstance)) {
    return "All accounts";
  }

  if (isPatternWrapper(connectorInstance)) {
    return connectorInstance.$pattern;
  }

  if (typeof connectorInstance !== "string") {
    return String(connectorInstance);
  }

  const selector = connectorInstance.trim();
  const byId = instances.find((i) => i.connector_instance_id === selector);
  if (byId?.display?.trim()) {
    return byId.display.trim();
  }

  const byDisplay = instances.filter((i) => i.display?.trim() === selector);
  if (byDisplay.length === 1) {
    const display = byDisplay[0]?.display?.trim();
    if (display) {
      return display;
    }
  }

  return selector;
}

export function connectorInstanceFromParameters(
  parameters: Record<string, unknown>,
): string {
  const raw = parameters[CONNECTOR_INSTANCE_PARAM];
  if (isAllAccountsConnectorInstance(raw)) {
    return "*";
  }
  if (typeof raw === "string") {
    const byId = raw.trim();
    return byId || "*";
  }
  return "*";
}

export function mergeConnectorInstanceIntoParameters(
  parameters: Record<string, unknown>,
  connectorInstance: string,
): Record<string, unknown> {
  if (parameters.$version === CONSTRAINT_VERSION && Array.isArray(parameters.groups)) {
    return mergeConnectorInstanceIntoStructured(parameters, connectorInstance);
  }
  const result = { ...parameters };
  if (connectorInstance === "*") {
    result[CONNECTOR_INSTANCE_PARAM] = "*";
  } else {
    result[CONNECTOR_INSTANCE_PARAM] = connectorInstance;
  }
  return result;
}

function mergeConnectorInstanceIntoStructured(
  constraints: Record<string, unknown>,
  connectorInstance: string,
): Record<string, unknown> {
  const groups = (constraints.groups as Array<Record<string, unknown>>).map(
    (group, index) => {
      if (index !== 0) return group;
      const conditions = [
        ...((group.conditions as Array<Record<string, unknown>>) ?? []),
        {
          field: CONNECTOR_INSTANCE_PARAM,
          op: "matches",
          value: connectorInstance === "*" ? "*" : connectorInstance,
        },
      ];
      return { ...group, conditions };
    },
  );
  return { ...constraints, groups };
}

const CONSTRAINT_VERSION = 2;

export function parametersWithoutConnectorInstance(
  parameters: Record<string, unknown>,
): Record<string, unknown> {
  const { [CONNECTOR_INSTANCE_PARAM]: _removed, ...rest } = parameters;
  return rest;
}

export function hasConnectorInstanceDisplayName(
  instance: AgentConnectorInstance,
): boolean {
  return !!instance.display?.trim();
}

export function selectableConnectorInstancesForAccount(
  instances: AgentConnectorInstance[],
  selectedInstanceId: string,
): AgentConnectorInstance[] {
  const keepSelected =
    selectedInstanceId !== "*" ? selectedInstanceId.trim() : "";

  return instances.filter(
    (instance) =>
      hasConnectorInstanceDisplayName(instance) ||
      instance.connector_instance_id === keepSelected,
  );
}

export function instanceSelectLabel(instance: AgentConnectorInstance): string {
  return instance.display?.trim() || "Unnamed account";
}

export function connectorInstanceFromStandingApprovalId(
  connectorInstanceId: string | null | undefined,
): string {
  if (!connectorInstanceId?.trim()) {
    return "*";
  }
  return connectorInstanceId.trim();
}

export function standingApprovalConnectorInstanceIdForUpdate(
  connectorInstance: string,
): string | null {
  return connectorInstance === "*" ? null : connectorInstance;
}
