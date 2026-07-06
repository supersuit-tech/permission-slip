import type { ActionConfiguration } from "../../hooks/useActionConfigs";
import type { ApprovalSummary } from "../../hooks/useApprovals";

const META_NAMESPACE_KEY = "$meta";
const DATA_WINDOW_NAMESPACE_KEY = "$data_window";

function isPatternWrapper(value: unknown): value is { $pattern: string } {
  return (
    typeof value === "object" &&
    value !== null &&
    "$pattern" in value &&
    typeof (value as Record<string, unknown>).$pattern === "string"
  );
}

function isWildcard(value: unknown): boolean {
  return value === "*";
}

function matchPattern(pattern: string, value: string): boolean {
  const parts = pattern.split("*");
  let regex = "^";
  for (let i = 0; i < parts.length; i++) {
    const part = parts[i] ?? "";
    regex += part.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    if (i < parts.length - 1) regex += ".*";
  }
  regex += "$";
  return new RegExp(regex).test(value);
}

function jsonValuesEqual(a: unknown, b: unknown): boolean {
  return JSON.stringify(a) === JSON.stringify(b);
}

function validateStringPattern(pattern: string, value: string): boolean {
  if (pattern.includes("*")) {
    return matchPattern(pattern, value);
  }
  return value === pattern;
}

function validateExecPatternConstraint(pattern: string, execValue: unknown): boolean {
  if (Array.isArray(execValue)) {
    if (execValue.length === 0) return false;
    return execValue.every(
      (value) => typeof value === "string" && validateStringPattern(pattern, value),
    );
  }
  return typeof execValue === "string" && validateStringPattern(pattern, execValue);
}

function validateRecipientPatternConstraint(pattern: string, sourceValue: unknown): boolean {
  if (!Array.isArray(sourceValue) || sourceValue.length === 0) return false;
  return sourceValue.some(
    (value) => typeof value === "string" && validateStringPattern(pattern, value),
  );
}

function flatMetaSourceValue(key: string, meta: Record<string, unknown>): unknown {
  switch (key) {
    case "sender":
      return meta.sender ?? meta.senders;
    case "from":
      return meta.from ?? meta.sender;
    default:
      return meta[key];
  }
}

function perMessageMetaSourceValue(key: string, msg: Record<string, unknown>): unknown {
  if (key === "sender" || key === "senders" || key === "from") {
    return msg.from;
  }
  return msg[key];
}

function isRecipientMetaKey(key: string): boolean {
  return key === "to" || key === "cc" || key === "bcc";
}

function validateConstraintValueAgainstSource(configValue: unknown, sourceValue: unknown): boolean {
  if (isWildcard(configValue)) return true;
  if (isPatternWrapper(configValue)) {
    if (sourceValue == null) return false;
    if (Array.isArray(sourceValue)) {
      return sourceValue.every(
        (value) => typeof value === "string" && validateStringPattern(configValue.$pattern, value),
      );
    }
    return typeof sourceValue === "string" && validateStringPattern(configValue.$pattern, sourceValue);
  }
  return sourceValue != null && jsonValuesEqual(configValue, sourceValue);
}

function validateMetaFieldConstraint(
  key: string,
  configValue: unknown,
  sourceValue: unknown,
): boolean {
  if (isRecipientMetaKey(key)) {
    if (isWildcard(configValue)) return true;
    if (isPatternWrapper(configValue)) {
      return validateRecipientPatternConstraint(configValue.$pattern, sourceValue);
    }
    if (!Array.isArray(sourceValue)) return false;
    return sourceValue.some((value) => value === configValue);
  }
  return validateConstraintValueAgainstSource(configValue, sourceValue);
}

function validateResolvedMetaConstraints(
  metaConstraints: Record<string, unknown>,
  resolvedMeta: Record<string, unknown>,
): boolean {
  const messages = resolvedMeta.messages;
  if (Array.isArray(messages) && messages.length > 0) {
    return messages.every((message) => {
      if (!message || typeof message !== "object" || Array.isArray(message)) return false;
      const msg = message as Record<string, unknown>;
      return Object.entries(metaConstraints).every(([key, configValue]) =>
        validateMetaFieldConstraint(key, configValue, perMessageMetaSourceValue(key, msg)),
      );
    });
  }

  return Object.entries(metaConstraints).every(([key, configValue]) =>
    validateMetaFieldConstraint(key, configValue, flatMetaSourceValue(key, resolvedMeta)),
  );
}

export function execParamsSatisfyConfigConstraints(
  configParams: Record<string, unknown>,
  execParams: Record<string, unknown>,
  resolvedMeta?: Record<string, unknown> | null,
): boolean {
  const config = { ...configParams };
  const metaConstraints = config[META_NAMESPACE_KEY];
  delete config[META_NAMESPACE_KEY];
  delete config[DATA_WINDOW_NAMESPACE_KEY];

  if (Object.keys(config).length === 0) {
    if (metaConstraints && typeof metaConstraints === "object" && !Array.isArray(metaConstraints)) {
      if (!resolvedMeta) return false;
      return validateResolvedMetaConstraints(
        metaConstraints as Record<string, unknown>,
        resolvedMeta,
      );
    }
    return true;
  }

  for (const [key, configValue] of Object.entries(config)) {
    if (isWildcard(configValue)) continue;

    if (isPatternWrapper(configValue)) {
      if (!(key in execParams)) return false;
      if (!validateExecPatternConstraint(configValue.$pattern, execParams[key])) return false;
      continue;
    }

    if (!(key in execParams)) return false;
    if (!jsonValuesEqual(configValue, execParams[key])) return false;
  }

  for (const key of Object.keys(execParams)) {
    if (!(key in config)) return false;
  }

  if (metaConstraints && typeof metaConstraints === "object" && !Array.isArray(metaConstraints)) {
    if (!resolvedMeta) return false;
    return validateResolvedMetaConstraints(
      metaConstraints as Record<string, unknown>,
      resolvedMeta,
    );
  }

  return true;
}

function configSpecificityScore(configParams: Record<string, unknown>): number {
  let score = 0;
  for (const [key, value] of Object.entries(configParams)) {
    if (key === DATA_WINDOW_NAMESPACE_KEY) {
      score += 2;
      continue;
    }
    if (key === META_NAMESPACE_KEY && value && typeof value === "object" && !Array.isArray(value)) {
      for (const metaValue of Object.values(value as Record<string, unknown>)) {
        if (!isWildcard(metaValue)) score += 2;
      }
      continue;
    }
    if (!isWildcard(value)) score += 1;
  }
  return score;
}

function normalizeEmailAddress(raw: string): string {
  const trimmed = raw.trim();
  const match = /<([^>]+)>/.exec(trimmed);
  return (match?.[1] ?? trimmed).trim();
}

function primarySenderAddress(from: unknown): string | null {
  if (typeof from === "string" && from.length > 0) return normalizeEmailAddress(from);
  if (Array.isArray(from)) {
    const first = from.find((value): value is string => typeof value === "string" && value.length > 0);
    return first ? normalizeEmailAddress(first) : null;
  }
  return null;
}

export function resourceDetailsToConstraintMeta(
  resourceDetails?: Record<string, unknown> | null,
): Record<string, unknown> | null {
  if (!resourceDetails) return null;

  const source =
    resourceDetails.in_reply_to && typeof resourceDetails.in_reply_to === "object"
      ? (resourceDetails.in_reply_to as Record<string, unknown>)
      : resourceDetails;

  const from = primarySenderAddress(source.from);
  if (!from) return null;

  return {
    sender: from,
    senders: [from],
    messages: [
      {
        from,
        to: source.to ?? [],
        cc: source.cc ?? [],
        bcc: source.bcc ?? [],
      },
    ],
  };
}

export function findBestMatchingActionConfig(
  configs: ReadonlyArray<Pick<ActionConfiguration, "id" | "status" | "action_type" | "parameters">>,
  actionType: string,
  execParams: Record<string, unknown>,
  resolvedMeta?: Record<string, unknown> | null,
): ActionConfiguration | null {
  const candidates = configs.filter(
    (config) => config.status === "active" && config.action_type === actionType,
  );

  let best: ActionConfiguration | null = null;
  let bestScore = -1;

  for (const config of candidates) {
    const params = (config.parameters ?? {}) as Record<string, unknown>;
    if (!execParamsSatisfyConfigConstraints(params, execParams, resolvedMeta)) {
      continue;
    }
    const score = configSpecificityScore(params);
    if (score > bestScore) {
      best = config as ActionConfiguration;
      bestScore = score;
    }
  }

  return best;
}

export function findMatchingActionConfigForApproval(
  configs: ReadonlyArray<ActionConfiguration>,
  approval: ApprovalSummary,
): ActionConfiguration | null {
  const params = approval.action.parameters as Record<string, unknown>;
  const resolvedMeta = resourceDetailsToConstraintMeta(
    approval.resource_details as Record<string, unknown> | undefined,
  );
  return findBestMatchingActionConfig(
    configs,
    approval.action.type,
    params,
    resolvedMeta,
  );
}
