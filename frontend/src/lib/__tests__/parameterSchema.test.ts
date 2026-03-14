import { describe, it, expect } from "vitest";
import {
  parseParametersSchema,
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
          enum: undefined,
          default: undefined,
          "x-ui": undefined,
        },
        repo: {
          type: "string",
          description: "Repository name",
          enum: undefined,
          default: undefined,
          "x-ui": undefined,
        },
        title: {
          type: "string",
          description: undefined,
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

    it("parses visible_when condition", () => {
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
      const widgets = ["text", "select", "textarea", "toggle", "number", "date"] as const;
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

    it("returns undefined x-ui for empty root x-ui", () => {
      const result = parseParametersSchema({
        type: "object",
        "x-ui": {},
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
          enum: undefined,
          default: undefined,
          "x-ui": undefined,
        },
      },
    });
  });
});
