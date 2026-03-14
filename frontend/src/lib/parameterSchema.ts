/**
 * Shared types and parsing for connector action parameter schemas.
 *
 * Both the approval review UI (SchemaParameterDetails) and the action
 * configuration form (ActionConfigParameterFields) consume parameter
 * schemas from the API. This module provides a single parsing function
 * to avoid duplication.
 */

/** Conditional visibility rule: show a field only when another field has a specific value. */
export interface VisibleWhen {
  field: string;
  equals: string | boolean | number;
}

/** Property-level `x-ui` rendering hints for a single parameter. */
export interface SchemaPropertyUI {
  widget?: "text" | "select" | "textarea" | "toggle" | "number" | "date";
  label?: string;
  placeholder?: string;
  group?: string;
  help_url?: string;
  help_text?: string;
  visible_when?: VisibleWhen;
}

/** A named group for organizing related fields in the form. */
export interface FieldGroup {
  id: string;
  label: string;
  description?: string;
  collapsed?: boolean;
}

/** Root-level `x-ui` rendering hints for the entire parameter schema. */
export interface SchemaUI {
  groups?: FieldGroup[];
  order?: string[];
}

/** JSON Schema property definition for a single action parameter. */
export interface SchemaProperty {
  type?: string;
  description?: string;
  enum?: string[];
  default?: unknown;
  "x-ui"?: SchemaPropertyUI;
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
  "x-ui"?: SchemaUI;
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
    return { type: "object", required, "x-ui": parseSchemaUI(schema["x-ui"]) };
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
        "x-ui": parsePropertyUI(prop["x-ui"]),
      };
    }
  }

  return { type: "object", required, properties, "x-ui": parseSchemaUI(schema["x-ui"]) };
}

/** All valid widget type values. */
export const VALID_WIDGETS = ["text", "select", "textarea", "toggle", "number", "date"] as const;

/** Widget type as a union — useful for exhaustive switch checks. */
export type WidgetType = (typeof VALID_WIDGETS)[number];

const VALID_WIDGETS_SET = new Set<string>(VALID_WIDGETS);

/**
 * Get the display label for a schema property.
 * Returns x-ui label if set, otherwise falls back to the property key.
 */
export function getFieldLabel(key: string, prop: SchemaProperty): string {
  return prop["x-ui"]?.label ?? key;
}

/**
 * Return property keys in the order specified by x-ui.order,
 * with any unmentioned keys appended in their original order.
 */
export function getOrderedFieldKeys(schema: ParametersSchema): string[] {
  const allKeys = Object.keys(schema.properties ?? {});
  const order = schema["x-ui"]?.order;
  if (!order || order.length === 0) return allKeys;

  const allKeysSet = new Set(allKeys);
  const ordered = order.filter((k) => allKeysSet.has(k));
  const orderedSet = new Set(ordered);
  const remaining = allKeys.filter((k) => !orderedSet.has(k));
  return [...ordered, ...remaining];
}

/**
 * Check whether a field should be visible given the current form values.
 * Returns true if the field has no visible_when rule or if the condition is met.
 */
export function isFieldVisible(
  prop: SchemaProperty,
  values: Record<string, unknown>,
): boolean {
  const rule = prop["x-ui"]?.visible_when;
  if (!rule) return true;
  return values[rule.field] === rule.equals;
}

/** Parse a property-level `x-ui` object, returning undefined if invalid. */
function parsePropertyUI(raw: unknown): SchemaPropertyUI | undefined {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return undefined;
  const obj = raw as Record<string, unknown>;

  const ui: SchemaPropertyUI = {};
  if (typeof obj.widget === "string" && VALID_WIDGETS_SET.has(obj.widget)) {
    ui.widget = obj.widget as SchemaPropertyUI["widget"];
  }
  if (typeof obj.label === "string") ui.label = obj.label;
  if (typeof obj.placeholder === "string") ui.placeholder = obj.placeholder;
  if (typeof obj.group === "string") ui.group = obj.group;
  if (typeof obj.help_url === "string") ui.help_url = obj.help_url;
  if (typeof obj.help_text === "string") ui.help_text = obj.help_text;
  if (obj.visible_when && typeof obj.visible_when === "object") {
    const vw = obj.visible_when as Record<string, unknown>;
    if (
      typeof vw.field === "string" &&
      (typeof vw.equals === "string" || typeof vw.equals === "boolean" || (typeof vw.equals === "number" && !Number.isNaN(vw.equals)))
    ) {
      ui.visible_when = { field: vw.field, equals: vw.equals as string | boolean | number };
    }
  }

  // Return undefined if no valid fields were parsed
  if (Object.keys(ui).length === 0) return undefined;
  return ui;
}

/** Parse a root-level `x-ui` object, returning undefined if invalid. */
function parseSchemaUI(raw: unknown): SchemaUI | undefined {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return undefined;
  const obj = raw as Record<string, unknown>;

  const ui: SchemaUI = {};

  if (Array.isArray(obj.groups)) {
    const groups: FieldGroup[] = [];
    for (const g of obj.groups) {
      if (g && typeof g === "object") {
        const group = g as Record<string, unknown>;
        if (typeof group.id === "string" && typeof group.label === "string") {
          const fg: FieldGroup = { id: group.id, label: group.label };
          if (typeof group.description === "string") fg.description = group.description;
          if (typeof group.collapsed === "boolean") fg.collapsed = group.collapsed;
          groups.push(fg);
        }
      }
    }
    if (groups.length > 0) ui.groups = groups;
  }

  if (Array.isArray(obj.order)) {
    const order = obj.order.filter((e): e is string => typeof e === "string");
    if (order.length > 0) ui.order = order;
  }

  if (Object.keys(ui).length === 0) return undefined;
  return ui;
}
