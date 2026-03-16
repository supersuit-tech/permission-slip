import { describe, it, expect } from "vitest";
import { buildParametersFromForm, type ParamMode } from "../ActionConfigFormFields";

describe("buildParametersFromForm", () => {
  it("auto-wraps value containing * as $pattern", () => {
    const result = buildParametersFromForm({ email: "*@company.com" });
    expect(result).toEqual({ email: { $pattern: "*@company.com" } });
  });

  it("auto-wraps prefix wildcard as $pattern", () => {
    const result = buildParametersFromForm({ repo: "supersuit-tech/*" });
    expect(result).toEqual({ repo: { $pattern: "supersuit-tech/*" } });
  });

  it("auto-wraps value with * even when mode is fixed", () => {
    const modes: Record<string, ParamMode> = { title: "fixed" };
    const result = buildParametersFromForm({ title: "Prefix: *" }, undefined, modes);
    expect(result).toEqual({ title: { $pattern: "Prefix: *" } });
  });

  it("stores plain string when value has no *", () => {
    const result = buildParametersFromForm({ name: "hello" });
    expect(result).toEqual({ name: "hello" });
  });

  it("stores wildcard mode as plain * string", () => {
    const modes: Record<string, ParamMode> = { title: "wildcard" };
    const result = buildParametersFromForm({ title: "*" }, undefined, modes);
    expect(result).toEqual({ title: "*" });
  });

  it("preserves legacy pattern mode for values without *", () => {
    const modes: Record<string, ParamMode> = { query: "pattern" };
    const result = buildParametersFromForm({ query: "exact-match" }, undefined, modes);
    expect(result).toEqual({ query: { $pattern: "exact-match" } });
  });

  it("skips empty values", () => {
    const result = buildParametersFromForm({ name: "", email: "test@example.com" });
    expect(result).toEqual({ email: "test@example.com" });
  });

  it("coerces integer types", () => {
    const schema = { count: { type: "integer" } };
    const result = buildParametersFromForm({ count: "42" }, schema);
    expect(result).toEqual({ count: 42 });
  });

  it("coerces number types", () => {
    const schema = { price: { type: "number" } };
    const result = buildParametersFromForm({ price: "3.14" }, schema);
    expect(result).toEqual({ price: 3.14 });
  });

  it("coerces boolean types", () => {
    const schema = { enabled: { type: "boolean" } };
    const result = buildParametersFromForm({ enabled: "true" }, schema);
    expect(result).toEqual({ enabled: true });
  });

  it("wildcard mode takes precedence over * auto-detect", () => {
    // Wildcard stores plain "*", not { $pattern: "*" }
    const modes: Record<string, ParamMode> = { field: "wildcard" };
    const result = buildParametersFromForm({ field: "*" }, undefined, modes);
    expect(result.field).toBe("*");
    expect(result.field).not.toEqual({ $pattern: "*" });
  });

  it("coerces array types from JSON string to array", () => {
    const schema = { tags: { type: "array" } };
    const result = buildParametersFromForm({ tags: '["a","b"]' }, schema);
    expect(result).toEqual({ tags: ["a", "b"] });
  });

  it("filters empty strings from array values at submission", () => {
    const schema = { tags: { type: "array" } };
    const result = buildParametersFromForm({ tags: '["a","","b",""]' }, schema);
    expect(result).toEqual({ tags: ["a", "b"] });
  });

  it("falls through to string when array JSON is invalid", () => {
    const schema = { tags: { type: "array" } };
    const result = buildParametersFromForm({ tags: "not-json" }, schema);
    expect(result).toEqual({ tags: "not-json" });
  });
});
