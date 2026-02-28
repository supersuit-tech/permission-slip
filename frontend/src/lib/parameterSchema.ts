/**
 * Shared types and parsing for connector action parameter schemas.
 *
 * Both the approval review UI (SchemaParameterDetails) and the action
 * configuration form (ActionConfigParameterFields) consume parameter
 * schemas from the API. This module provides a single parsing function
 * to avoid duplication.
 */

/** JSON Schema property definition for a single action parameter. */
export interface SchemaProperty {
  type?: string;
  description?: string;
  enum?: string[];
  default?: unknown;
}

/**
 * Minimal JSON Schema for action parameters. Matches the `parameters_schema`
 * stored on connector actions — a JSON Schema object with properties and
 * optional required list.
 */
export interface ParametersSchema {
  type: string;
  required?: string[];
  properties?: Record<string, SchemaProperty>;
}

/**
 * Parse a parameters_schema from the API (typed as Record<string, unknown>)
 * into a typed ParametersSchema, or null if not valid.
 *
 * Validates the top-level shape and type-narrows each property field
 * individually to avoid unsafe casts.
 */
export function parseParametersSchema(
  schema: Record<string, unknown> | undefined | null,
): ParametersSchema | null {
  if (!schema) return null;
  if (typeof schema !== "object") return null;
  if (schema.type !== "object") return null;

  const required = Array.isArray(schema.required)
    ? schema.required.filter((e): e is string => typeof e === "string")
    : undefined;

  const rawProps = schema.properties;
  if (!rawProps || typeof rawProps !== "object") {
    return { type: "object", required };
  }

  const properties: Record<string, SchemaProperty> = {};
  for (const [key, val] of Object.entries(
    rawProps as Record<string, unknown>,
  )) {
    if (typeof val === "object" && val !== null) {
      const prop = val as Record<string, unknown>;
      properties[key] = {
        type: typeof prop.type === "string" ? prop.type : undefined,
        description:
          typeof prop.description === "string" ? prop.description : undefined,
        enum: Array.isArray(prop.enum)
          ? prop.enum.filter((e): e is string => typeof e === "string")
          : undefined,
        default: prop.default,
      };
    }
  }

  return { type: "object", required, properties };
}
