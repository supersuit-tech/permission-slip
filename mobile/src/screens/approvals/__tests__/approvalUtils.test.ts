import {
  secondsUntil,
  formatCountdown,
  humanizeActionType,
  buildActionSummary,
  formatRelativeTime,
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
