import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import {
  ActionPreviewSummary,
  buildSummary,
} from "../ActionPreviewSummary";
import type { ParametersSchema } from "@/lib/parameterSchema";

// Shorthand for curly quotes used by renderPlain
function q(s: string) {
  return `\u201C${s}\u201D`;
}

// ---------------------------------------------------------------------------
// buildSummary – action-type-specific formatters
// ---------------------------------------------------------------------------

describe("buildSummary", () => {
  describe("github.create_issue", () => {
    it("renders owner/repo and title", () => {
      expect(
        buildSummary(
          "github.create_issue",
          { owner: "acme", repo: "widgets", title: "Fix login bug" },
          null,
          "Create Issue",
        ),
      ).toBe(`Create issue ${q("Fix login bug")} in ${q("acme/widgets")}`);
    });

    it("omits repo reference when owner/repo missing", () => {
      expect(
        buildSummary(
          "github.create_issue",
          { title: "Fix login bug" },
          null,
          "Create Issue",
        ),
      ).toBe(`Create issue ${q("Fix login bug")}`);
    });

    it("falls back to generic when title missing", () => {
      const result = buildSummary(
        "github.create_issue",
        { owner: "acme", repo: "widgets" },
        null,
        "Create Issue",
      );
      expect(result).toContain("Create Issue");
    });
  });

  describe("github.merge_pr", () => {
    it("renders PR number and repo", () => {
      expect(
        buildSummary(
          "github.merge_pr",
          { owner: "acme", repo: "widgets", pull_number: 42 },
          null,
          "Merge Pull Request",
        ),
      ).toBe(`Merge PR ${q("#42")} in ${q("acme/widgets")}`);
    });

    it("shows merge method when non-default", () => {
      expect(
        buildSummary(
          "github.merge_pr",
          {
            owner: "acme",
            repo: "widgets",
            pull_number: 7,
            merge_method: "squash",
          },
          null,
          "Merge Pull Request",
        ),
      ).toBe(`Merge PR ${q("#7")} in ${q("acme/widgets")} (squash)`);
    });

    it("omits merge method when it is the default 'merge'", () => {
      expect(
        buildSummary(
          "github.merge_pr",
          {
            owner: "acme",
            repo: "widgets",
            pull_number: 7,
            merge_method: "merge",
          },
          null,
          "Merge Pull Request",
        ),
      ).toBe(`Merge PR ${q("#7")} in ${q("acme/widgets")}`);
    });
  });

  describe("slack.send_message", () => {
    it("renders channel and truncated message", () => {
      const result = buildSummary(
        "slack.send_message",
        { channel: "#general", message: "Hello team!" },
        null,
        "Send Message",
      );
      expect(result).toBe(
        `Send message to ${q("#general")} \u2014 Hello team!`,
      );
    });

    it("renders channel only when no message", () => {
      expect(
        buildSummary(
          "slack.send_message",
          { channel: "#ops" },
          null,
          "Send Message",
        ),
      ).toBe(`Send message to ${q("#ops")}`);
    });

    it("truncates long messages", () => {
      const longMsg = "A".repeat(100);
      const result = buildSummary(
        "slack.send_message",
        { channel: "#general", message: longMsg },
        null,
        "Send Message",
      );
      expect(result.length).toBeLessThan(130);
      expect(result).toContain("\u2026");
    });
  });

  describe("slack.create_channel", () => {
    it("renders public channel", () => {
      expect(
        buildSummary(
          "slack.create_channel",
          { name: "new-project" },
          null,
          "Create Channel",
        ),
      ).toBe(`Create channel ${q("#new-project")}`);
    });

    it("renders private channel", () => {
      expect(
        buildSummary(
          "slack.create_channel",
          { name: "secret-ops", is_private: true },
          null,
          "Create Channel",
        ),
      ).toBe(`Create private channel ${q("#secret-ops")}`);
    });
  });

  describe("email.send", () => {
    it("renders single recipient and subject", () => {
      expect(
        buildSummary(
          "email.send",
          { to: "bob@example.com", subject: "Meeting tomorrow" },
          null,
          null,
        ),
      ).toBe(
        `Send email to ${q("bob@example.com")} with subject ${q("Meeting tomorrow")}`,
      );
    });

    it("renders array of recipients", () => {
      const result = buildSummary(
        "email.send",
        {
          to: ["alice@example.com", "bob@example.com"],
          subject: "Update",
        },
        null,
        null,
      );
      expect(result).toContain("alice@example.com, bob@example.com");
    });

    it("summarizes many recipients", () => {
      const result = buildSummary(
        "email.send",
        {
          to: ["a@x.com", "b@x.com", "c@x.com", "d@x.com"],
          subject: "All hands",
        },
        null,
        null,
      );
      expect(result).toContain("and 2 more");
    });
  });

  describe("payment.charge", () => {
    it("renders amount in dollars", () => {
      expect(
        buildSummary(
          "payment.charge",
          { amount: 9900, currency: "USD", description: "Monthly subscription" },
          null,
          null,
        ),
      ).toBe(
        `Charge ${q("$99.00")} for ${q("Monthly subscription")}`,
      );
    });

    it("renders EUR symbol", () => {
      const result = buildSummary(
        "payment.charge",
        { amount: 5000, currency: "EUR" },
        null,
        null,
      );
      expect(result).toBe(`Charge ${q("\u20AC50.00")}`);
    });
  });

  describe("generic / unknown action types", () => {
    it("uses actionName when available", () => {
      const schema: ParametersSchema = {
        type: "object",
        required: ["target"],
        properties: {
          target: { type: "string", description: "Target resource" },
        },
      };
      const result = buildSummary(
        "custom.deploy",
        { target: "production" },
        schema,
        "Deploy Application",
      );
      expect(result).toContain("Deploy Application");
      expect(result).toContain("production");
    });

    it("humanizes action type when no actionName", () => {
      const result = buildSummary(
        "data.export_csv",
        { format: "csv" },
        null,
        null,
      );
      // humanizeActionType extracts the operation (last segment) only,
      // avoiding naive capitalization of connector names
      expect(result).toContain("Export csv");
    });

    it("shows up to 3 highlighted params", () => {
      const schema: ParametersSchema = {
        type: "object",
        required: ["a", "b"],
        properties: {
          a: { type: "string", description: "First" },
          b: { type: "string", description: "Second" },
          c: { type: "string", description: "Third" },
          d: { type: "string", description: "Fourth" },
        },
      };
      const result = buildSummary(
        "test.action",
        { a: "1", b: "2", c: "3", d: "4" },
        schema,
        null,
      );
      expect(result).toContain("First");
      expect(result).toContain("Second");
      expect(result).toContain("Third");
      expect(result).not.toContain("Fourth");
    });

    it("returns just the label when no params", () => {
      expect(buildSummary("test.noop", {}, null, "Do Nothing")).toBe(
        "Do Nothing",
      );
    });
  });
});

// ---------------------------------------------------------------------------
// ActionPreviewSummary – rendering tests
// ---------------------------------------------------------------------------

describe("ActionPreviewSummary", () => {
  it("renders a paragraph with the summary text", () => {
    render(
      <ActionPreviewSummary
        actionType="github.create_issue"
        parameters={{ owner: "acme", repo: "web", title: "Bug" }}
        schema={null}
        actionName="Create Issue"
      />,
    );

    const el = screen.getByTestId("action-preview-summary");
    expect(el.tagName).toBe("P");
    expect(el.textContent).toContain("Bug");
    expect(el.textContent).toContain("acme/web");
  });

  it("renders highlighted values with ValSpan styling", () => {
    render(
      <ActionPreviewSummary
        actionType="github.create_issue"
        parameters={{ owner: "acme", repo: "web", title: "Bug" }}
        schema={null}
        actionName="Create Issue"
      />,
    );

    // ValSpan wraps values in a span with bg-muted class
    const highlighted = screen.getByTestId("action-preview-summary")
      .querySelectorAll("span");
    expect(highlighted.length).toBeGreaterThanOrEqual(1);
  });

  it("renders generic summary for unknown action type", () => {
    render(
      <ActionPreviewSummary
        actionType="custom.process"
        parameters={{ input: "file.csv" }}
        schema={null}
        actionName="Process Data"
      />,
    );

    expect(screen.getByTestId("action-preview-summary").textContent).toContain(
      "Process Data",
    );
  });
});
