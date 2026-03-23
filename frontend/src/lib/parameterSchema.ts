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
  widget?:
    | "text"
    | "select"
    | "remote-select"
    | "textarea"
    | "toggle"
    | "number"
    | "date"
    | "datetime"
    | "list";
  label?: string;
  placeholder?: string;
  group?: string;
  help_url?: string;
  help_text?: string;
  /** Path template for openapi-fetch, e.g. `/v1/agents/{agent_id}/connectors/{connector_id}/calendars`. */
  remote_select_options_path?: string;
  /** JSON key for option value (default `id`). */
  remote_select_id_key?: string;
  /** JSON key for option label (default `summary`, then `name`). */
  remote_select_label_key?: string;
  /** Placeholder for manual entry when user toggles fallback. */
  remote_select_fallback_placeholder?: string;
  visible_when?: VisibleWhen;
  /**
   * Sibling parameter key for datetime range pairing (e.g. time_min ↔ time_max).
   * Use with `datetime_range_role` so the sibling's fixed value sets HTML min/max on this input.
   */
  datetime_range_pair?: string;
  /** Whether this field is the lower or upper bound of the pair. */
  datetime_range_role?: "lower" | "upper";
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
  format?: string;
  description?: string;
  enum?: string[];
  default?: unknown;
  items?: { type?: string };
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
      const propType = typeof prop.type === "string" ? prop.type : undefined;
      const format = typeof prop.format === "string" ? prop.format : undefined;
      const description = typeof prop.description === "string" ? prop.description : undefined;
      const parsedUI = parsePropertyUI(prop["x-ui"]);
      const items = parseItems(prop.items);
      // Auto-map types to widgets when no explicit widget is set
      const ui = autoMapWidget(propType, format, items, description, parsedUI);
      properties[key] = {
        type: propType,
        format,
        description,
        enum: Array.isArray(prop.enum)
          ? prop.enum.filter((e): e is string => typeof e === "string")
          : undefined,
        default: prop.default,
        items,
        "x-ui": ui,
      };
    }
  }

  return { type: "object", required, properties, "x-ui": parseSchemaUI(schema["x-ui"]) };
}

/** All valid widget type values. */
export const VALID_WIDGETS = [
  "text",
  "select",
  "remote-select",
  "textarea",
  "toggle",
  "number",
  "date",
  "datetime",
  "list",
] as const;

/** Widget type as a union — useful for exhaustive switch checks. */
export type WidgetType = (typeof VALID_WIDGETS)[number];

const VALID_WIDGETS_SET = new Set<string>(VALID_WIDGETS);

/**
 * Get the display label for a schema property.
 * Returns x-ui label if set, otherwise falls back to the property key.
 */
export function getFieldLabel(key: string, prop: SchemaProperty): string {
  if (prop["x-ui"]?.label) return prop["x-ui"].label;
  return key
    .replace(/_/g, " ")
    .replace(/\b\w/g, (c) => c.toUpperCase())
    .replace(/\bId\b/g, "ID")
    .replace(/\bUrl\b/g, "URL")
    .replace(/\bApi\b/g, "API");
}

/**
 * Return property keys in the order specified by x-ui.order,
 * with any unmentioned keys appended in their original order.
 */
export function getOrderedFieldKeys(schema: ParametersSchema): string[] {
  const allKeys = Object.keys(schema.properties ?? {});
  const order = schema["x-ui"]?.order;

  if (order && order.length > 0) {
    const allKeysSet = new Set(allKeys);
    const ordered = order.filter((k) => allKeysSet.has(k));
    const orderedSet = new Set(ordered);
    const remaining = allKeys.filter((k) => !orderedSet.has(k));
    return [...ordered, ...remaining];
  }

  // No explicit order: required fields first, then optional in original order
  const required = new Set(schema.required ?? []);
  return [
    ...allKeys.filter((k) => required.has(k)),
    ...allKeys.filter((k) => !required.has(k)),
  ];
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
  const fieldValue = values[rule.field];
  // Form values are always strings, but rule.equals may be boolean or number.
  // Coerce via String() so "true" matches true and "42" matches 42.
  if (typeof rule.equals === "boolean") {
    return String(fieldValue) === String(rule.equals);
  }
  if (typeof rule.equals === "number") {
    return Number(fieldValue) === rule.equals;
  }
  return fieldValue === rule.equals;
}

const FRIENDLY_TYPE_LABELS: Record<string, string> = {
  string: "text",
  boolean: "yes / no",
  integer: "number",
  number: "number",
  array: "list",
};

/**
 * Return a user-friendly label for a JSON Schema type.
 * Returns undefined for types that should not be displayed (object, unknown).
 */
export function friendlyTypeLabel(type: string | undefined): string | undefined {
  if (!type) return undefined;
  return FRIENDLY_TYPE_LABELS[type];
}

/**
 * Infer the appropriate widget for a schema property based on its type/format.
 * Returns "text" as default when no better match is found.
 */
export function inferWidgetFromProperty(property: SchemaProperty): WidgetType {
  if (property.format === "date-time") return "datetime";
  if (property.type === "boolean") return "toggle";
  if (property.type === "integer" || property.type === "number") return "number";
  if (property.type === "array" && (!property.items || property.items?.type === "string")) return "list";

  // Heuristic: detect datetime fields by description mentioning RFC 3339 or ISO 8601
  if (property.type === "string" && property.description) {
    const desc = property.description.toLowerCase();
    if ((desc.includes("rfc 3339") || desc.includes("rfc3339") || desc.includes("iso 8601")) && !desc.includes("epoch")) {
      return "datetime";
    }
    // Heuristic: detect textarea fields for markdown body content
    if (desc.includes("markdown")) {
      return "textarea";
    }
  }

  return "text";
}

/**
 * Auto-map JSON Schema type/format to a widget when no explicit widget is set.
 * Explicit x-ui.widget always takes precedence. Uses inferWidgetFromProperty
 * for the actual mapping logic.
 */
function autoMapWidget(
  type: string | undefined,
  format: string | undefined,
  items: { type?: string } | undefined,
  description: string | undefined,
  parsedUI: SchemaPropertyUI | undefined,
): SchemaPropertyUI | undefined {
  if (parsedUI?.widget) return parsedUI;

  const inferred = inferWidgetFromProperty({ type, format, items, description });
  if (inferred !== "text") {
    return { ...parsedUI, widget: inferred };
  }
  return parsedUI;
}

/** Parse an items schema from a JSON Schema array property. */
function parseItems(raw: unknown): { type?: string } | undefined {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return undefined;
  const obj = raw as Record<string, unknown>;
  if (typeof obj.type === "string") return { type: obj.type };
  return undefined;
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
  if (typeof obj.remote_select_options_path === "string" && obj.remote_select_options_path.length > 0) {
    ui.remote_select_options_path = obj.remote_select_options_path;
  }
  if (typeof obj.remote_select_id_key === "string" && obj.remote_select_id_key.length > 0) {
    ui.remote_select_id_key = obj.remote_select_id_key;
  }
  if (typeof obj.remote_select_label_key === "string" && obj.remote_select_label_key.length > 0) {
    ui.remote_select_label_key = obj.remote_select_label_key;
  }
  if (typeof obj.remote_select_fallback_placeholder === "string") {
    ui.remote_select_fallback_placeholder = obj.remote_select_fallback_placeholder;
  }
  if (obj.visible_when && typeof obj.visible_when === "object") {
    const vw = obj.visible_when as Record<string, unknown>;
    if (
      typeof vw.field === "string" &&
      (typeof vw.equals === "string" || typeof vw.equals === "boolean" || (typeof vw.equals === "number" && !Number.isNaN(vw.equals)))
    ) {
      ui.visible_when = { field: vw.field, equals: vw.equals as string | boolean | number };
    }
  }

  if (typeof obj.datetime_range_pair === "string" && obj.datetime_range_pair.length > 0) {
    ui.datetime_range_pair = obj.datetime_range_pair;
  }
  if (obj.datetime_range_role === "lower" || obj.datetime_range_role === "upper") {
    ui.datetime_range_role = obj.datetime_range_role;
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
