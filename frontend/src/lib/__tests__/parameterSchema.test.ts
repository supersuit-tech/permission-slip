import { describe, it, expect } from "vitest";
import {
  parseParametersSchema,
  getFieldLabel,
  getOrderedFieldKeys,
  isFieldVisible,
  inferWidgetFromProperty,
  type SchemaPropertyUI,
  type SchemaUI,
} from "../parameterSchema";

describe("parseParametersSchema", () => {
  it("parses a valid schema with properties", () => {
    const result = parseParametersSchema({
      type: "object",
      required: ["owner", "repo"],
      properties: {
        owner: { type: "string", description: "Repository owner" },
        repo: { type: "string", description: "Repository name" },
        title: { type: "string" },
      },
    });

    expect(result).toEqual({
      type: "object",
      required: ["owner", "repo"],
      "x-ui": undefined,
      properties: {
        owner: {
          type: "string",
          description: "Repository owner",
          format: undefined,
          enum: undefined,
          default: undefined,
          "x-ui": undefined,
        },
        repo: {
          type: "string",
          description: "Repository name",
          format: undefined,
          enum: undefined,
          default: undefined,
          "x-ui": undefined,
        },
        title: {
          type: "string",
          description: undefined,
          format: undefined,
          enum: undefined,
          default: undefined,
          "x-ui": undefined,
        },
      },
    });
  });

  it("parses enum and default values", () => {
    const result = parseParametersSchema({
      type: "object",
      properties: {
        method: {
          type: "string",
          enum: ["merge", "squash", "rebase"],
          default: "merge",
          description: "Merge strategy",
        },
      },
    });

    expect(result?.properties?.method).toEqual({
      type: "string",
      format: undefined,
      description: "Merge strategy",
      enum: ["merge", "squash", "rebase"],
      default: "merge",
      "x-ui": undefined,
    });
  });

  it("filters non-string enum values", () => {
    const result = parseParametersSchema({
      type: "object",
      properties: {
        level: {
          type: "string",
          enum: ["low", 42, "high", null, "medium"],
        },
      },
    });

    expect(result?.properties?.level?.enum).toEqual([
      "low",
      "high",
      "medium",
    ]);
  });

  it("returns null for undefined input", () => {
    expect(parseParametersSchema(undefined)).toBeNull();
  });

  it("returns null for null input", () => {
    expect(parseParametersSchema(null)).toBeNull();
  });

  it("returns null for non-object type", () => {
    expect(parseParametersSchema({ type: "array" })).toBeNull();
  });

  it("returns schema without properties when properties field is missing", () => {
    const result = parseParametersSchema({
      type: "object",
      required: ["foo"],
    });

    expect(result).toEqual({
      type: "object",
      required: ["foo"],
      "x-ui": undefined,
    });
  });

  it("skips non-object property values", () => {
    const result = parseParametersSchema({
      type: "object",
      properties: {
        valid: { type: "string" },
        invalid: "not an object",
        alsoInvalid: null,
      },
    });

    expect(result?.properties).toEqual({
      valid: {
        type: "string",
        description: undefined,
        enum: undefined,
        default: undefined,
        "x-ui": undefined,
      },
    });
  });

  it("handles missing required array", () => {
    const result = parseParametersSchema({
      type: "object",
      properties: { a: { type: "string" } },
    });

    expect(result?.required).toBeUndefined();
  });

  it("handles non-string type and description fields", () => {
    const result = parseParametersSchema({
      type: "object",
      properties: {
        weird: { type: 123, description: true },
      },
    });

    expect(result?.properties?.weird).toEqual({
      type: undefined,
      format: undefined,
      description: undefined,
      enum: undefined,
      default: undefined,
      "x-ui": undefined,
    });
  });

  describe("x-ui property-level parsing", () => {
    it("parses all property-level x-ui fields", () => {
      const result = parseParametersSchema({
        type: "object",
        properties: {
          currency: {
            type: "string",
            enum: ["usd", "eur", "gbp"],
            "x-ui": {
              widget: "select",
              label: "Currency",
              placeholder: "Choose currency",
              group: "billing",
              help_url: "https://example.com/help",
              help_text: "Pick the billing currency",
            },
          },
        },
      });

      const ui = result?.properties?.currency?.["x-ui"];
      expect(ui).toEqual({
        widget: "select",
        label: "Currency",
        placeholder: "Choose currency",
        group: "billing",
        help_url: "https://example.com/help",
        help_text: "Pick the billing currency",
      } satisfies SchemaPropertyUI);
    });

    it("parses visible_when condition with string equals", () => {
      const result = parseParametersSchema({
        type: "object",
        properties: {
          details: {
            type: "string",
            "x-ui": {
              visible_when: { field: "mode", equals: "advanced" },
            },
          },
        },
      });

      expect(result?.properties?.details?.["x-ui"]?.visible_when).toEqual({
        field: "mode",
        equals: "advanced",
      });
    });

    it("parses visible_when condition with boolean equals", () => {
      const result = parseParametersSchema({
        type: "object",
        properties: {
          details: {
            type: "string",
            "x-ui": {
              visible_when: { field: "enabled", equals: true },
            },
          },
        },
      });

      expect(result?.properties?.details?.["x-ui"]?.visible_when).toEqual({
        field: "enabled",
        equals: true,
      });
    });

    it("parses visible_when condition with number equals", () => {
      const result = parseParametersSchema({
        type: "object",
        properties: {
          details: {
            type: "string",
            "x-ui": {
              visible_when: { field: "priority", equals: 1 },
            },
          },
        },
      });

      expect(result?.properties?.details?.["x-ui"]?.visible_when).toEqual({
        field: "priority",
        equals: 1,
      });
    });

    it("ignores invalid widget values", () => {
      const result = parseParametersSchema({
        type: "object",
        properties: {
          field: {
            type: "string",
            "x-ui": { widget: "invalid_widget", label: "Valid Label" },
          },
        },
      });

      const ui = result?.properties?.field?.["x-ui"];
      expect(ui?.widget).toBeUndefined();
      expect(ui?.label).toBe("Valid Label");
    });

    it("returns undefined x-ui for empty x-ui object", () => {
      const result = parseParametersSchema({
        type: "object",
        properties: {
          field: { type: "string", "x-ui": {} },
        },
      });

      expect(result?.properties?.field?.["x-ui"]).toBeUndefined();
    });

    it("returns undefined x-ui when x-ui is not an object", () => {
      const result = parseParametersSchema({
        type: "object",
        properties: {
          field: { type: "string", "x-ui": "not-an-object" },
        },
      });

      expect(result?.properties?.field?.["x-ui"]).toBeUndefined();
    });

    it("rejects NaN as visible_when equals value", () => {
      const result = parseParametersSchema({
        type: "object",
        properties: {
          field: {
            type: "string",
            "x-ui": { visible_when: { field: "priority", equals: NaN } },
          },
        },
      });

      expect(result?.properties?.field?.["x-ui"]).toBeUndefined();
    });

    it("returns undefined x-ui when x-ui is an array", () => {
      const result = parseParametersSchema({
        type: "object",
        properties: {
          field: { type: "string", "x-ui": ["not", "an", "object"] },
        },
      });

      expect(result?.properties?.field?.["x-ui"]).toBeUndefined();
    });

    it("ignores visible_when with missing fields", () => {
      const result = parseParametersSchema({
        type: "object",
        properties: {
          field: {
            type: "string",
            "x-ui": { visible_when: { field: "mode" } },
          },
        },
      });

      // visible_when is invalid (missing equals), so x-ui is empty → undefined
      expect(result?.properties?.field?.["x-ui"]).toBeUndefined();
    });

    it("accepts all valid widget types", () => {
      const widgets = ["text", "select", "textarea", "toggle", "number", "date", "datetime"] as const;
      for (const widget of widgets) {
        const result = parseParametersSchema({
          type: "object",
          properties: {
            field: { type: "string", "x-ui": { widget } },
          },
        });
        expect(result?.properties?.field?.["x-ui"]?.widget).toBe(widget);
      }
    });
  });

  describe("x-ui root-level parsing", () => {
    it("parses groups and order", () => {
      const result = parseParametersSchema({
        type: "object",
        "x-ui": {
          groups: [
            { id: "billing", label: "Billing", description: "Payment fields" },
            { id: "options", label: "Options", collapsed: true },
          ],
          order: ["customer_id", "currency", "auto_advance"],
        },
        properties: {
          customer_id: { type: "string" },
          currency: { type: "string" },
          auto_advance: { type: "boolean" },
        },
      });

      const ui = result?.["x-ui"];
      expect(ui).toEqual({
        groups: [
          { id: "billing", label: "Billing", description: "Payment fields" },
          { id: "options", label: "Options", collapsed: true },
        ],
        order: ["customer_id", "currency", "auto_advance"],
      } satisfies SchemaUI);
    });

    it("returns undefined x-ui when not present", () => {
      const result = parseParametersSchema({
        type: "object",
        properties: { a: { type: "string" } },
      });

      expect(result?.["x-ui"]).toBeUndefined();
    });

    it("skips groups with missing id or label", () => {
      const result = parseParametersSchema({
        type: "object",
        "x-ui": {
          groups: [
            { id: "valid", label: "Valid" },
            { id: "missing-label" },
            { label: "Missing ID" },
            "not-an-object",
          ],
        },
        properties: { a: { type: "string" } },
      });

      expect(result?.["x-ui"]?.groups).toEqual([{ id: "valid", label: "Valid" }]);
    });

    it("filters non-string values from order array", () => {
      const result = parseParametersSchema({
        type: "object",
        "x-ui": {
          order: ["field_a", 42, "field_b", null],
        },
        properties: { field_a: { type: "string" }, field_b: { type: "string" } },
      });

      expect(result?.["x-ui"]?.order).toEqual(["field_a", "field_b"]);
    });

    it("preserves root-level x-ui when properties field is missing", () => {
      const result = parseParametersSchema({
        type: "object",
        "x-ui": {
          groups: [{ id: "main", label: "Main" }],
          order: ["field_a"],
        },
      });

      expect(result?.["x-ui"]?.groups).toEqual([{ id: "main", label: "Main" }]);
      expect(result?.["x-ui"]?.order).toEqual(["field_a"]);
      expect(result?.properties).toBeUndefined();
    });

    it("returns undefined x-ui for empty root x-ui", () => {
      const result = parseParametersSchema({
        type: "object",
        "x-ui": {},
        properties: { a: { type: "string" } },
      });

      expect(result?.["x-ui"]).toBeUndefined();
    });

    it("returns undefined x-ui when root x-ui is an array", () => {
      const result = parseParametersSchema({
        type: "object",
        "x-ui": ["not", "valid"],
        properties: { a: { type: "string" } },
      });

      expect(result?.["x-ui"]).toBeUndefined();
    });
  });

  it("parses full schema with both root and property x-ui", () => {
    const input = {
      type: "object",
      required: ["customer_id"],
      "x-ui": {
        groups: [
          { id: "billing", label: "Billing" },
          { id: "options", label: "Options", collapsed: true },
        ],
        order: ["customer_id", "currency", "auto_advance"],
      },
      properties: {
        customer_id: {
          type: "string",
          "x-ui": { label: "Customer", placeholder: "cus_ABC123", group: "billing" },
        },
        currency: {
          type: "string",
          enum: ["usd", "eur", "gbp"],
          "x-ui": { widget: "select", group: "billing" },
        },
        auto_advance: {
          type: "boolean",
          "x-ui": { widget: "toggle", label: "Auto-send invoice", group: "options" },
        },
      },
    };

    const result = parseParametersSchema(input);

    expect(result?.["x-ui"]?.groups).toHaveLength(2);
    expect(result?.["x-ui"]?.order).toEqual(["customer_id", "currency", "auto_advance"]);
    expect(result?.properties?.customer_id?.["x-ui"]?.label).toBe("Customer");
    expect(result?.properties?.currency?.["x-ui"]?.widget).toBe("select");
    expect(result?.properties?.auto_advance?.["x-ui"]?.widget).toBe("toggle");
  });

  describe("getFieldLabel", () => {
    it("returns x-ui label when set", () => {
      expect(getFieldLabel("customer_id", { type: "string", "x-ui": { label: "Customer" } })).toBe("Customer");
    });

    it("converts snake_case key to Title Case when no x-ui label", () => {
      expect(getFieldLabel("customer_id", { type: "string" })).toBe("Customer ID");
    });

    it("converts key to Title Case when x-ui exists but no label", () => {
      expect(getFieldLabel("amount", { type: "number", "x-ui": { widget: "number" } })).toBe("Amount");
    });
  });

  describe("getOrderedFieldKeys", () => {
    it("returns keys in x-ui order with remaining appended", () => {
      const schema = parseParametersSchema({
        type: "object",
        "x-ui": { order: ["b", "a"] },
        properties: { a: { type: "string" }, b: { type: "string" }, c: { type: "string" } },
      })!;
      expect(getOrderedFieldKeys(schema)).toEqual(["b", "a", "c"]);
    });

    it("returns original order when no x-ui order", () => {
      const schema = parseParametersSchema({
        type: "object",
        properties: { x: { type: "string" }, y: { type: "string" } },
      })!;
      expect(getOrderedFieldKeys(schema)).toEqual(["x", "y"]);
    });

    it("skips order keys that don't exist in properties", () => {
      const schema = parseParametersSchema({
        type: "object",
        "x-ui": { order: ["missing", "a"] },
        properties: { a: { type: "string" }, b: { type: "string" } },
      })!;
      expect(getOrderedFieldKeys(schema)).toEqual(["a", "b"]);
    });

    it("handles schema with no properties", () => {
      const schema = parseParametersSchema({ type: "object" })!;
      expect(getOrderedFieldKeys(schema)).toEqual([]);
    });
  });

  describe("isFieldVisible", () => {
    it("returns true when no visible_when rule", () => {
      expect(isFieldVisible({ type: "string" }, {})).toBe(true);
    });

    it("returns true when condition is met (string)", () => {
      const prop = { type: "string", "x-ui": { visible_when: { field: "mode", equals: "advanced" as string | boolean | number } } };
      expect(isFieldVisible(prop, { mode: "advanced" })).toBe(true);
    });

    it("returns false when condition is not met", () => {
      const prop = { type: "string", "x-ui": { visible_when: { field: "mode", equals: "advanced" as string | boolean | number } } };
      expect(isFieldVisible(prop, { mode: "basic" })).toBe(false);
    });

    it("returns false when referenced field is missing", () => {
      const prop = { type: "string", "x-ui": { visible_when: { field: "mode", equals: "advanced" as string | boolean | number } } };
      expect(isFieldVisible(prop, {})).toBe(false);
    });

    it("works with boolean condition", () => {
      const prop = { type: "string", "x-ui": { visible_when: { field: "enabled", equals: true as string | boolean | number } } };
      expect(isFieldVisible(prop, { enabled: true })).toBe(true);
      expect(isFieldVisible(prop, { enabled: false })).toBe(false);
    });
  });

  describe("format field and datetime auto-mapping", () => {
    it("extracts format field from property", () => {
      const result = parseParametersSchema({
        type: "object",
        properties: {
          start_time: { type: "string", format: "date-time" },
        },
      });

      expect(result?.properties?.start_time?.format).toBe("date-time");
    });

    it("auto-maps format date-time to datetime widget when no explicit widget", () => {
      const result = parseParametersSchema({
        type: "object",
        properties: {
          start_time: { type: "string", format: "date-time" },
        },
      });

      expect(result?.properties?.start_time?.["x-ui"]?.widget).toBe("datetime");
    });

    it("does not override explicit widget when format is date-time", () => {
      const result = parseParametersSchema({
        type: "object",
        properties: {
          start_time: {
            type: "string",
            format: "date-time",
            "x-ui": { widget: "text" },
          },
        },
      });

      expect(result?.properties?.start_time?.["x-ui"]?.widget).toBe("text");
    });

    it("sets format to undefined when not present", () => {
      const result = parseParametersSchema({
        type: "object",
        properties: {
          name: { type: "string" },
        },
      });

      expect(result?.properties?.name?.format).toBeUndefined();
    });
  });

  describe("inferWidgetFromProperty heuristics", () => {
    it("infers datetime for RFC 3339 description", () => {
      expect(inferWidgetFromProperty({
        type: "string",
        description: "Start time in RFC 3339 format",
      })).toBe("datetime");
    });

    it("infers datetime for ISO 8601 description", () => {
      expect(inferWidgetFromProperty({
        type: "string",
        description: "Due date in ISO 8601 format",
      })).toBe("datetime");
    });

    it("does not infer datetime when description mentions epoch", () => {
      expect(inferWidgetFromProperty({
        type: "string",
        description: "Start date/time (ISO 8601 or epoch milliseconds)",
      })).toBe("text");
    });

    it("infers textarea for markdown description", () => {
      expect(inferWidgetFromProperty({
        type: "string",
        description: "Issue body (Markdown supported)",
      })).toBe("textarea");
    });

    it("infers textarea for lowercase markdown description", () => {
      expect(inferWidgetFromProperty({
        type: "string",
        description: "Comment body (markdown)",
      })).toBe("textarea");
    });

    it("does not apply heuristics for non-string types", () => {
      expect(inferWidgetFromProperty({
        type: "integer",
        description: "Some RFC 3339 thing",
      })).toBe("number");
    });
  });

  describe("auto-mapping heuristics in parseParametersSchema", () => {
    it("auto-maps RFC 3339 string fields to datetime widget", () => {
      const result = parseParametersSchema({
        type: "object",
        properties: {
          start_time: {
            type: "string",
            description: "Start time in RFC 3339 format (e.g. '2024-01-15T09:00:00-05:00')",
          },
        },
      });

      expect(result?.properties?.start_time?.["x-ui"]?.widget).toBe("datetime");
    });

    it("auto-maps markdown body fields to textarea widget", () => {
      const result = parseParametersSchema({
        type: "object",
        properties: {
          body: {
            type: "string",
            description: "Issue body (Markdown supported)",
          },
        },
      });

      expect(result?.properties?.body?.["x-ui"]?.widget).toBe("textarea");
    });
  });

  it("preserves backwards compatibility — schemas without x-ui parse identically", () => {
    const input = {
      type: "object",
      required: ["owner"],
      properties: {
        owner: { type: "string", description: "Repository owner" },
      },
    };

    const result = parseParametersSchema(input);

    expect(result).toEqual({
      type: "object",
      required: ["owner"],
      "x-ui": undefined,
      properties: {
        owner: {
          type: "string",
          description: "Repository owner",
          format: undefined,
          enum: undefined,
          default: undefined,
          "x-ui": undefined,
        },
      },
    });
  });
});
