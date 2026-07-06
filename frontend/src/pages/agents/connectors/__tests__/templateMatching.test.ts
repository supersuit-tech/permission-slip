import { describe, it, expect } from "vitest";
import {
  deepEqualJSON,
  templateIsApplied,
  templateMatchesStandingApproval,
} from "../templateMatching";
import type { StandingApproval } from "@/hooks/useStandingApprovals";
import type { StandingApprovalTemplate } from "@/hooks/useStandingApprovalTemplates";

function makeTemplate(
  overrides: Partial<StandingApprovalTemplate> = {},
): StandingApprovalTemplate {
  return {
    id: "tpl_test",
    connector_id: "github",
    action_type: "github.create_issue",
    name: "Test template",
    description: null,
    constraints: { repo: "supersuit-tech/webapp", title: "*", body: "*" },
    created_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

function makeRule(
  overrides: Partial<StandingApproval> = {},
): StandingApproval {
  return {
    standing_approval_id: "sa_test",
    agent_id: 42,
    user_id: "user-1",
    action_type: "github.create_issue",
    action_version: "1",
    constraints: { repo: "supersuit-tech/webapp", title: "*", body: "*" },
    status: "active",
    starts_at: "2026-01-01T00:00:00Z",
    created_at: "2026-01-01T00:00:00Z",
    ...overrides,
  } as StandingApproval;
}

describe("deepEqualJSON", () => {
  it("returns true for identical primitives", () => {
    expect(deepEqualJSON("a", "a")).toBe(true);
    expect(deepEqualJSON(1, 1)).toBe(true);
    expect(deepEqualJSON(true, true)).toBe(true);
    expect(deepEqualJSON(null, null)).toBe(true);
  });

  it("is order-insensitive for object keys", () => {
    expect(deepEqualJSON({ a: 1, b: 2 }, { b: 2, a: 1 })).toBe(true);
  });
});

describe("templateMatchesStandingApproval", () => {
  it("matches when action_type and constraints are equal", () => {
    const tpl = makeTemplate();
    const rule = makeRule();
    expect(templateMatchesStandingApproval(tpl, rule)).toBe(true);
  });

  it("does not match when action_types differ", () => {
    const tpl = makeTemplate({ action_type: "github.create_issue" });
    const rule = makeRule({ action_type: "github.close_issue" });
    expect(templateMatchesStandingApproval(tpl, rule)).toBe(false);
  });

  it("never matches a wildcard rule action type", () => {
    const tpl = makeTemplate({ constraints: {} });
    const rule = makeRule({ action_type: "*", constraints: {} });
    expect(templateMatchesStandingApproval(tpl, rule)).toBe(false);
  });
});

describe("templateIsApplied", () => {
  it("returns true when any active rule matches", () => {
    const tpl = makeTemplate();
    const rules = [
      makeRule({
        action_type: "github.merge_pr",
        constraints: { repo: "x", pr: 1 },
      }),
      makeRule(),
    ];
    expect(templateIsApplied(tpl, rules)).toBe(true);
  });

  it("returns false when only revoked rules match", () => {
    const tpl = makeTemplate();
    const rules = [makeRule({ status: "revoked" })];
    expect(templateIsApplied(tpl, rules)).toBe(false);
  });

  it("returns false for empty rules", () => {
    expect(templateIsApplied(makeTemplate(), [])).toBe(false);
  });
});
