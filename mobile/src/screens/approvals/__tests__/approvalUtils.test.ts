import {
  secondsUntil,
  formatCountdown,
  humanizeActionType,
  buildActionSummary,
  formatRelativeTime,
  safeParams,
  isExpired,
  formatParamValue,
  formatTimestamp,
} from "../approvalUtils";

describe("secondsUntil", () => {
  it("returns positive seconds for future time", () => {
    const future = new Date(Date.now() + 120_000).toISOString();
    const result = secondsUntil(future);
    expect(result).toBeGreaterThanOrEqual(119);
    expect(result).toBeLessThanOrEqual(120);
  });

  it("returns 0 for past time", () => {
    const past = new Date(Date.now() - 10_000).toISOString();
    expect(secondsUntil(past)).toBe(0);
  });
});

describe("formatCountdown", () => {
  it("formats zero seconds", () => {
    expect(formatCountdown(0)).toBe("0:00");
  });

  it("formats seconds with padding", () => {
    expect(formatCountdown(5)).toBe("0:05");
  });

  it("formats minutes and seconds", () => {
    expect(formatCountdown(125)).toBe("2:05");
  });

  it("formats exact minutes", () => {
    expect(formatCountdown(300)).toBe("5:00");
  });
});

describe("humanizeActionType", () => {
  it("capitalizes and spaces the operation", () => {
    expect(humanizeActionType("github.create_issue")).toBe("Create issue");
  });

  it("handles single segment", () => {
    expect(humanizeActionType("deploy")).toBe("Deploy");
  });

  it("handles multi-segment (reverse DNS)", () => {
    expect(humanizeActionType("com.example.deploy.production")).toBe(
      "Production",
    );
  });
});

describe("buildActionSummary", () => {
  it("formats github.create_issue", () => {
    const result = buildActionSummary("github.create_issue", {
      owner: "acme",
      repo: "widgets",
      title: "Fix bug",
    });
    expect(result).toContain("Create issue");
    expect(result).toContain("Fix bug");
    expect(result).toContain("acme/widgets");
  });

  it("formats email.send", () => {
    const result = buildActionSummary("email.send", {
      to: ["alice@example.com"],
      subject: "Hello",
    });
    expect(result).toContain("Send email to");
    expect(result).toContain("alice@example.com");
  });

  it("formats slack.send_message", () => {
    const result = buildActionSummary("slack.send_message", {
      channel: "#general",
      message: "Hello team",
    });
    expect(result).toContain("Send message to");
    expect(result).toContain("#general");
  });

  it("falls back to generic summary for unknown types", () => {
    const result = buildActionSummary("custom.do_thing", {
      target: "prod",
    });
    expect(result).toContain("Do thing");
    expect(result).toContain("target");
    expect(result).toContain("prod");
  });

  it("returns humanized label for empty parameters", () => {
    expect(buildActionSummary("email.send", {})).toBe("Send");
  });
});

describe("formatRelativeTime", () => {
  it("shows 'Just now' for very recent times", () => {
    const now = new Date(Date.now() - 5_000).toISOString();
    expect(formatRelativeTime(now)).toBe("Just now");
  });

  it("shows minutes for times < 1 hour ago", () => {
    const fiveMinAgo = new Date(Date.now() - 5 * 60_000).toISOString();
    expect(formatRelativeTime(fiveMinAgo)).toBe("5m ago");
  });

  it("shows hours for times < 24 hours ago", () => {
    const threeHrAgo = new Date(Date.now() - 3 * 3600_000).toISOString();
    expect(formatRelativeTime(threeHrAgo)).toBe("3h ago");
  });

  it("shows days for times < 7 days ago", () => {
    const twoDaysAgo = new Date(Date.now() - 2 * 86400_000).toISOString();
    expect(formatRelativeTime(twoDaysAgo)).toBe("2d ago");
  });

  it("shows date for older times", () => {
    const oldDate = new Date(Date.now() - 30 * 86400_000).toISOString();
    const result = formatRelativeTime(oldDate);
    // Should contain a month abbreviation, not "ago"
    expect(result).not.toContain("ago");
  });
});

describe("safeParams", () => {
  it("returns the object when given a plain object", () => {
    const obj = { key: "value" };
    expect(safeParams(obj)).toBe(obj);
  });

  it("returns empty object for null", () => {
    expect(safeParams(null)).toEqual({});
  });

  it("returns empty object for undefined", () => {
    expect(safeParams(undefined)).toEqual({});
  });

  it("returns empty object for an array", () => {
    expect(safeParams([1, 2, 3])).toEqual({});
  });

  it("returns empty object for a string", () => {
    expect(safeParams("hello")).toEqual({});
  });
});

describe("isExpired", () => {
  it("returns true for pending approval past expiry", () => {
    const past = new Date(Date.now() - 10_000).toISOString();
    expect(isExpired("pending", past)).toBe(true);
  });

  it("returns false for pending approval not yet expired", () => {
    const future = new Date(Date.now() + 300_000).toISOString();
    expect(isExpired("pending", future)).toBe(false);
  });

  it("returns false for non-pending status even if past expiry", () => {
    const past = new Date(Date.now() - 10_000).toISOString();
    expect(isExpired("approved", past)).toBe(false);
  });
});

describe("formatParamValue", () => {
  it("formats null", () => {
    expect(formatParamValue(null)).toBe("null");
  });

  it("formats strings", () => {
    expect(formatParamValue("hello")).toBe("hello");
  });

  it("formats numbers", () => {
    expect(formatParamValue(42)).toBe("42");
  });

  it("formats booleans", () => {
    expect(formatParamValue(true)).toBe("true");
  });

  it("formats arrays", () => {
    expect(formatParamValue(["a", "b"])).toBe("a, b");
  });

  it("formats objects as JSON", () => {
    const result = formatParamValue({ nested: true });
    expect(result).toContain('"nested": true');
  });
});

describe("formatTimestamp", () => {
  it("formats a valid ISO string", () => {
    const result = formatTimestamp("2026-01-15T14:30:00Z");
    // Should contain some date representation
    expect(result.length).toBeGreaterThan(0);
  });

  it("returns input string for invalid date", () => {
    expect(formatTimestamp("not-a-date")).toBe("not-a-date");
  });
});
