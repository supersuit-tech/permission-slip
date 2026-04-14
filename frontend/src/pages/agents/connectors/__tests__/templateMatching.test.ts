import { describe, it, expect } from "vitest";
import {
  deepEqualJSON,
  templateIsApplied,
  templateMatchesConfig,
} from "../templateMatching";
import type { ActionConfiguration } from "../../../../hooks/useActionConfigs";
import type { ActionConfigTemplate } from "../../../../hooks/useActionConfigTemplates";

function makeTemplate(
  overrides: Partial<ActionConfigTemplate> = {},
): ActionConfigTemplate {
  return {
    id: "tpl_test",
    connector_id: "github",
    action_type: "github.create_issue",
    name: "Test template",
    description: null,
    parameters: { repo: "supersuit-tech/webapp", title: "*", body: "*" },
    created_at: "2026-01-01T00:00:00Z",
    ...overrides,
  } as ActionConfigTemplate;
}

function makeConfig(
  overrides: Partial<ActionConfiguration> = {},
): ActionConfiguration {
  return {
    id: "ac_test",
    agent_id: 42,
    connector_id: "github",
    action_type: "github.create_issue",
    parameters: { repo: "supersuit-tech/webapp", title: "*", body: "*" },
    status: "active",
    name: "Something",
    description: null,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  } as ActionConfiguration;
}

describe("deepEqualJSON", () => {
  it("returns true for identical primitives", () => {
    expect(deepEqualJSON("a", "a")).toBe(true);
    expect(deepEqualJSON(1, 1)).toBe(true);
    expect(deepEqualJSON(true, true)).toBe(true);
    expect(deepEqualJSON(null, null)).toBe(true);
  });

  it("returns false for different primitives", () => {
    expect(deepEqualJSON("a", "b")).toBe(false);
    expect(deepEqualJSON(1, 2)).toBe(false);
    expect(deepEqualJSON(true, false)).toBe(false);
    expect(deepEqualJSON(null, undefined)).toBe(false);
    expect(deepEqualJSON(null, 0)).toBe(false);
    expect(deepEqualJSON(1, "1")).toBe(false);
  });

  it("is order-insensitive for object keys", () => {
    expect(deepEqualJSON({ a: 1, b: 2 }, { b: 2, a: 1 })).toBe(true);
  });

  it("is order-sensitive for arrays", () => {
    expect(deepEqualJSON([1, 2, 3], [1, 2, 3])).toBe(true);
    expect(deepEqualJSON([1, 2, 3], [3, 2, 1])).toBe(false);
  });

  it("distinguishes arrays from objects", () => {
    expect(deepEqualJSON([], {})).toBe(false);
  });

  it("handles nested objects", () => {
    expect(
      deepEqualJSON(
        { a: { b: { c: [1, 2] } } },
        { a: { b: { c: [1, 2] } } },
      ),
    ).toBe(true);
    expect(
      deepEqualJSON(
        { a: { b: { c: [1, 2] } } },
        { a: { b: { c: [1, 3] } } },
      ),
    ).toBe(false);
  });

  it("returns false when one side has extra keys", () => {
    expect(deepEqualJSON({ a: 1 }, { a: 1, b: 2 })).toBe(false);
    expect(deepEqualJSON({ a: 1, b: 2 }, { a: 1 })).toBe(false);
  });

  it("equates $pattern wrappers by content", () => {
    expect(
      deepEqualJSON({ $pattern: "supersuit-tech/*" }, { $pattern: "supersuit-tech/*" }),
    ).toBe(true);
    expect(
      deepEqualJSON({ $pattern: "foo/*" }, { $pattern: "bar/*" }),
    ).toBe(false);
  });
});

describe("templateMatchesConfig", () => {
  it("matches when action_type and parameters are equal", () => {
    const tpl = makeTemplate();
    const cfg = makeConfig();
    expect(templateMatchesConfig(tpl, cfg)).toBe(true);
  });

  it("matches regardless of parameter key order", () => {
    const tpl = makeTemplate({ parameters: { a: 1, b: 2 } });
    const cfg = makeConfig({ parameters: { b: 2, a: 1 } });
    expect(templateMatchesConfig(tpl, cfg)).toBe(true);
  });

  it("does not match when action_types differ", () => {
    const tpl = makeTemplate({ action_type: "github.create_issue" });
    const cfg = makeConfig({ action_type: "github.close_issue" });
    expect(templateMatchesConfig(tpl, cfg)).toBe(false);
  });

  it("does not match when a parameter value differs", () => {
    const tpl = makeTemplate({
      parameters: { repo: "supersuit-tech/webapp", title: "*" },
    });
    const cfg = makeConfig({
      parameters: { repo: "supersuit-tech/api", title: "*" },
    });
    expect(templateMatchesConfig(tpl, cfg)).toBe(false);
  });

  it("does not match when config has an extra parameter", () => {
    const tpl = makeTemplate({ parameters: { repo: "r" } });
    const cfg = makeConfig({ parameters: { repo: "r", label: "bug" } });
    expect(templateMatchesConfig(tpl, cfg)).toBe(false);
  });

  it("does not match when template has an extra parameter", () => {
    const tpl = makeTemplate({ parameters: { repo: "r", label: "bug" } });
    const cfg = makeConfig({ parameters: { repo: "r" } });
    expect(templateMatchesConfig(tpl, cfg)).toBe(false);
  });

  it("matches $pattern parameter wrappers on both sides", () => {
    const tpl = makeTemplate({
      parameters: { repo: { $pattern: "supersuit-tech/*" }, title: "*" },
    });
    const cfg = makeConfig({
      parameters: { repo: { $pattern: "supersuit-tech/*" }, title: "*" },
    });
    expect(templateMatchesConfig(tpl, cfg)).toBe(true);
  });

  it("does not match when $pattern values differ", () => {
    const tpl = makeTemplate({
      parameters: { repo: { $pattern: "supersuit-tech/*" } },
    });
    const cfg = makeConfig({
      parameters: { repo: { $pattern: "other-org/*" } },
    });
    expect(templateMatchesConfig(tpl, cfg)).toBe(false);
  });

  it("never matches a wildcard config", () => {
    const tpl = makeTemplate({ parameters: {} });
    const cfg = makeConfig({ action_type: "*", parameters: {} });
    expect(templateMatchesConfig(tpl, cfg)).toBe(false);
  });

  it("ignores name, description, status, and timestamps", () => {
    const tpl = makeTemplate();
    const cfg = makeConfig({
      name: "Renamed by user",
      description: "Custom description",
      status: "disabled",
    });
    expect(templateMatchesConfig(tpl, cfg)).toBe(true);
  });
});

describe("templateIsApplied", () => {
  it("returns true when any config matches", () => {
    const tpl = makeTemplate();
    const configs = [
      makeConfig({
        action_type: "github.merge_pr",
        parameters: { repo: "x", pr: 1 },
      }),
      makeConfig(),
    ];
    expect(templateIsApplied(tpl, configs)).toBe(true);
  });

  it("returns false when no configs match", () => {
    const tpl = makeTemplate();
    const configs = [
      makeConfig({
        action_type: "github.merge_pr",
        parameters: { repo: "x", pr: 1 },
      }),
    ];
    expect(templateIsApplied(tpl, configs)).toBe(false);
  });

  it("returns false for empty configs", () => {
    expect(templateIsApplied(makeTemplate(), [])).toBe(false);
  });
});
