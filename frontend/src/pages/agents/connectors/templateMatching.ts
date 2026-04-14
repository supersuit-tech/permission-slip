import type { ActionConfiguration } from "@/hooks/useActionConfigs";
import type { ActionConfigTemplate } from "@/hooks/useActionConfigTemplates";

/**
 * Stable deep-equality for JSON-shaped values. Ignores object key order so
 * that `{a: 1, b: 2}` equals `{b: 2, a: 1}`. Arrays are order-sensitive.
 */
export function deepEqualJSON(a: unknown, b: unknown): boolean {
  if (Object.is(a, b)) return true;
  if (a === null || b === null) return a === b;
  if (typeof a !== typeof b) return false;
  if (typeof a !== "object") return false;

  const aIsArray = Array.isArray(a);
  const bIsArray = Array.isArray(b);
  if (aIsArray !== bIsArray) return false;
  if (aIsArray && bIsArray) {
    if (a.length !== b.length) return false;
    for (let i = 0; i < a.length; i++) {
      if (!deepEqualJSON(a[i], b[i])) return false;
    }
    return true;
  }

  const ao = a as Record<string, unknown>;
  const bo = b as Record<string, unknown>;
  const aKeys = Object.keys(ao);
  const bKeys = Object.keys(bo);
  if (aKeys.length !== bKeys.length) return false;
  for (const k of aKeys) {
    if (!Object.prototype.hasOwnProperty.call(bo, k)) return false;
    if (!deepEqualJSON(ao[k], bo[k])) return false;
  }
  return true;
}

/**
 * True iff `config` represents the same permission scope as `template`:
 * same `action_type` and deep-equal `parameters`. Ignores name, description,
 * status, and approval mode (those don't affect the constraint scope).
 *
 * Wildcard configs (`action_type === "*"`) never match any template — they
 * are the "Enable All" escape hatch, not a template equivalent.
 */
export function templateMatchesConfig(
  template: Pick<ActionConfigTemplate, "action_type" | "parameters">,
  config: Pick<ActionConfiguration, "action_type" | "parameters">,
): boolean {
  if (config.action_type === "*") return false;
  if (config.action_type !== template.action_type) return false;
  return deepEqualJSON(template.parameters, config.parameters);
}

/**
 * True iff any config in `configs` is an equivalent of `template`.
 */
export function templateIsApplied(
  template: Pick<ActionConfigTemplate, "action_type" | "parameters">,
  configs: ReadonlyArray<
    Pick<ActionConfiguration, "action_type" | "parameters">
  >,
): boolean {
  return configs.some((c) => templateMatchesConfig(template, c));
}
