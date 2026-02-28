import { describe, it, expect } from "vitest";
import { parseParametersSchema } from "../parameterSchema";

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
      properties: {
        owner: {
          type: "string",
          description: "Repository owner",
          enum: undefined,
          default: undefined,
        },
        repo: {
          type: "string",
          description: "Repository name",
          enum: undefined,
          default: undefined,
        },
        title: {
          type: "string",
          description: undefined,
          enum: undefined,
          default: undefined,
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
    });
  });
});
