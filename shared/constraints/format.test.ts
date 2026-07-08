import {
  formatStandingApprovalConstraints,
  formatStandingApprovalConstraintsText,
} from "./format";

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

  it("labels $meta from constraints as verified sender", () => {
    const lines = formatStandingApprovalConstraints({
      message_id: "*",
      $meta: { from: { $pattern: "auto-confirm@amazon.com" } },
    });
    expect(lines).toEqual([
      {
        label: "message_id",
        mode: "wildcard",
        value: "any",
        verified: false,
      },
      {
        label: "Verified sender",
        mode: "pattern",
        value: "auto-confirm@amazon.com",
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

  it("formats readable multiline text", () => {
    const text = formatStandingApprovalConstraintsText({
      $meta: { to: "team@mycorp.com" },
    });
    expect(text).toBe("Verified To: team@mycorp.com");
  });

  it("formats $data_window last_days", () => {
    const lines = formatStandingApprovalConstraints({
      chat_id: "42",
      $data_window: { last_days: 30 },
    });
    expect(lines).toContainEqual({
      label: "Data window",
      mode: "fixed",
      value: "last 30 days",
      verified: false,
    });
  });

  it("formats comparison operators", () => {
    const lines = formatStandingApprovalConstraints({
      $version: 2,
      match: "any",
      groups: [
        {
          match: "all",
          conditions: [{ field: "limit", op: "lte", value: 20 }],
        },
      ],
    });
    expect(lines).toEqual([
      {
        label: "limit",
        mode: "fixed",
        value: "20",
        verified: false,
        comparisonOp: "lte",
      },
    ]);
    expect(
      formatStandingApprovalConstraintsText({
        $version: 2,
        match: "any",
        groups: [
          {
            match: "all",
            conditions: [{ field: "limit", op: "lte", value: 20 }],
          },
        ],
      }),
    ).toBe("limit: at most 20");
  });
});
