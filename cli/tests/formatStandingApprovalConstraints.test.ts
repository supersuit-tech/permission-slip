import {
  formatStandingApprovalConstraints,
  formatStandingApprovalConstraintsText,
  parseConstraintsJson,
} from "../src/formatStandingApprovalConstraints.js";

describe("formatStandingApprovalConstraints", () => {
  it("formats legacy flat constraints", () => {
    const lines = formatStandingApprovalConstraints({
      recipient: { $pattern: "*@acme.com" },
      $meta: { from: "boss@acme.com" },
    });

    expect(lines).toEqual([
      {
        label: "recipient",
        mode: "pattern",
        value: "*@acme.com",
        verified: false,
      },
      {
        label: "Verified sender",
        mode: "fixed",
        value: "boss@acme.com",
        verified: true,
      },
    ]);
  });

  it("formats v2 structured constraints with negation and scenarios", () => {
    const lines = formatStandingApprovalConstraints({
      $version: 2,
      match: "any",
      groups: [
        {
          match: "all",
          conditions: [
            {
              field: "recipient",
              op: "any_of",
              values: ["*@acme.com", "boss@partner.com"],
            },
            {
              field: "$meta.from",
              op: "none_of",
              values: ["ceo@acme.com"],
            },
          ],
        },
        {
          match: "all",
          conditions: [{ field: "channel", op: "matches", value: "general" }],
        },
      ],
    });

    expect(lines).toEqual([
      {
        label: "Scenario 1: recipient",
        mode: "fixed",
        value: "*@acme.com",
        verified: false,
        negated: false,
      },
      {
        label: "Scenario 1: recipient",
        mode: "fixed",
        value: "boss@partner.com",
        verified: false,
        negated: false,
      },
      {
        label: "Scenario 1: Verified sender",
        mode: "fixed",
        value: "ceo@acme.com",
        verified: true,
        negated: true,
      },
      {
        label: "Scenario 2: channel",
        mode: "fixed",
        value: "general",
        verified: false,
        negated: false,
      },
    ]);
  });

  it("builds plain-text summary", () => {
    const text = formatStandingApprovalConstraintsText({
      recipient: "*",
      $meta: { from: { $pattern: "*@company.com" } },
    });

    expect(text).toBe(
      "recipient: any value\nVerified sender: *@company.com",
    );
  });

  it("parseConstraintsJson rejects invalid JSON", () => {
    expect(() => parseConstraintsJson("not-json")).toThrow(
      "--constraints must be valid JSON",
    );
  });

  it("parseConstraintsJson rejects non-object values", () => {
    expect(() => parseConstraintsJson('["a"]')).toThrow(
      "--constraints must be a JSON object",
    );
  });
});
