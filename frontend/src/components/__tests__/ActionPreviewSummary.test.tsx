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

  describe("slack.send_message with resourceDetails", () => {
    it("uses channel_name from resourceDetails", () => {
      const result = buildSummary(
        "slack.send_message",
        { channel: "C0AMRGKRTA4", message: "Hello!" },
        null,
        "Send Message",
        undefined,
        { channel_name: "#general" },
      );
      expect(result).toContain(q("#general"));
      expect(result).not.toContain("C0AMRGKRTA4");
    });

    it("falls back to raw channel when no resourceDetails", () => {
      const result = buildSummary(
        "slack.send_message",
        { channel: "C0AMRGKRTA4", message: "Hello!" },
        null,
        "Send Message",
      );
      expect(result).toContain(q("C0AMRGKRTA4"));
    });
  });

  describe("slack.send_dm", () => {
    it("uses user_name from resourceDetails", () => {
      const result = buildSummary(
        "slack.send_dm",
        { user_id: "U12345678", message: "Hey!" },
        null,
        "Send DM",
        undefined,
        { user_name: "Johnny" },
      );
      expect(result).toContain(q("Johnny"));
      expect(result).not.toContain("U12345678");
    });

    it("falls back to raw user_id when no resourceDetails", () => {
      const result = buildSummary(
        "slack.send_dm",
        { user_id: "U12345678", message: "Hey!" },
        null,
        "Send DM",
      );
      expect(result).toContain(q("U12345678"));
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

  // ── Google Calendar ─────────────────────────────────────────────

  describe("google.delete_calendar_event", () => {
    it("shows event title and time from resource details", () => {
      const result = buildSummary(
        "google.delete_calendar_event",
        { event_id: "evt123" },
        null,
        null,
        undefined,
        { title: "Q1 Planning", start_time: "2026-03-15T14:00:00Z" },
      );
      expect(result).toContain("Delete event");
      expect(result).toContain("Q1 Planning");
    });

    it("falls back to generic when no resource details", () => {
      const result = buildSummary(
        "google.delete_calendar_event",
        { event_id: "evt123" },
        null,
        "Delete Calendar Event",
      );
      expect(result).toContain("Delete Calendar Event");
    });
  });

  describe("google.update_calendar_event", () => {
    it("shows event title from resource details", () => {
      const result = buildSummary(
        "google.update_calendar_event",
        { event_id: "evt123", summary: "New Title" },
        null,
        null,
        undefined,
        { title: "Old Title", start_time: "2026-03-15T14:00:00Z" },
      );
      expect(result).toContain("Update event");
      expect(result).toContain("Old Title");
    });
  });

  // ── Google Drive ────────────────────────────────────────────────

  describe("google.delete_drive_file", () => {
    it("shows file name from resource details", () => {
      const result = buildSummary(
        "google.delete_drive_file",
        { file_id: "f123" },
        null,
        null,
        undefined,
        { file_name: "Budget 2026.xlsx" },
      );
      expect(result).toBe(`Delete file ${q("Budget 2026.xlsx")}`);
    });
  });

  // ── Google Docs ─────────────────────────────────────────────────

  describe("google.get_document", () => {
    it("shows document title from resource details", () => {
      const result = buildSummary(
        "google.get_document",
        { document_id: "doc123" },
        null,
        null,
        undefined,
        { title: "Project Spec" },
      );
      expect(result).toBe(`Get document ${q("Project Spec")}`);
    });
  });

  // ── Google Sheets ───────────────────────────────────────────────

  describe("google.sheets_read_range", () => {
    it("shows spreadsheet name and range", () => {
      const result = buildSummary(
        "google.sheets_read_range",
        { spreadsheet_id: "s123", range: "Sheet1!A1:B5" },
        null,
        null,
        undefined,
        { title: "Budget Tracker", range: "Sheet1!A1:B5" },
      );
      expect(result).toContain("Read");
      expect(result).toContain("Budget Tracker");
      expect(result).toContain("Sheet1!A1:B5");
    });
  });

  // ── Google Slides ───────────────────────────────────────────────

  describe("google.get_presentation", () => {
    it("shows presentation title", () => {
      const result = buildSummary(
        "google.get_presentation",
        { presentation_id: "p123" },
        null,
        null,
        undefined,
        { title: "Q1 Review Deck" },
      );
      expect(result).toBe(`Get presentation ${q("Q1 Review Deck")}`);
    });
  });

  // ── Gmail ───────────────────────────────────────────────────────

  describe("google.read_email", () => {
    it("shows subject and sender", () => {
      const result = buildSummary(
        "google.read_email",
        { message_id: "msg123" },
        null,
        null,
        undefined,
        { subject: "Weekly Update", from: "alice@example.com" },
      );
      expect(result).toContain("Read email");
      expect(result).toContain("Weekly Update");
      expect(result).toContain("alice@example.com");
    });
  });

  describe("google.archive_email", () => {
    it("shows subject and sender", () => {
      const result = buildSummary(
        "google.archive_email",
        { thread_id: "t123" },
        null,
        null,
        undefined,
        { subject: "Old Thread", from: "bob@example.com" },
      );
      expect(result).toContain("Archive email");
      expect(result).toContain("Old Thread");
      expect(result).toContain("bob@example.com");
    });
  });

  describe("google.send_email_reply", () => {
    it("shows original subject", () => {
      const result = buildSummary(
        "google.send_email_reply",
        { message_id: "msg456" },
        null,
        null,
        undefined,
        { subject: "Re: Budget Discussion" },
      );
      expect(result).toBe(`Reply to ${q("Re: Budget Discussion")}`);
    });
  });

  // ── Proton Mail ─────────────────────────────────────────────────

  describe("protonmail.send_email", () => {
    it("shows recipients and subject", () => {
      const result = buildSummary(
        "protonmail.send_email",
        {
          to: ["alice@example.com", "bob@example.com"],
          subject: "Project update",
          body: "Hello",
        },
        null,
        "Send Email",
      );
      expect(result).toContain("Send email to");
      expect(result).toContain("alice@example.com");
      expect(result).toContain("Project update");
    });
  });

  describe("protonmail.read_inbox", () => {
    it("shows folder and limit", () => {
      const result = buildSummary(
        "protonmail.read_inbox",
        { folder: "INBOX", limit: 10 },
        null,
        "Read Inbox",
      );
      expect(result).toContain("Read");
      expect(result).toContain("10");
      expect(result).toContain("INBOX");
    });

    it("notes unread-only filter", () => {
      const result = buildSummary(
        "protonmail.read_inbox",
        { folder: "INBOX", limit: 5, unread_only: true },
        null,
        "Read Inbox",
      );
      expect(result).toContain("unread only");
    });
  });

  describe("protonmail.search_emails", () => {
    it("shows folder and search filters", () => {
      const result = buildSummary(
        "protonmail.search_emails",
        { folder: "INBOX", subject: "invoice", from: "acme.com" },
        null,
        "Search Emails",
      );
      expect(result).toContain("Search");
      expect(result).toContain("INBOX");
      expect(result).toContain("invoice");
      expect(result).toContain("acme.com");
    });
  });

  describe("protonmail.read_email", () => {
    it("shows subject and sender from resourceDetails", () => {
      const result = buildSummary(
        "protonmail.read_email",
        { message_id: 10, folder: "INBOX" },
        null,
        null,
        undefined,
        { subject: "Weekly Update", from: ["alice@example.com"] },
      );
      expect(result).toContain("Read email");
      expect(result).toContain("Weekly Update");
      expect(result).toContain("alice@example.com");
    });

    it("falls back when enrichment is absent", () => {
      const result = buildSummary(
        "protonmail.read_email",
        { message_id: 10, folder: "INBOX" },
        null,
        "Read Email",
      );
      expect(result).toContain("Read Email");
      expect(result).toContain("10");
    });
  });

  describe("protonmail.archive_email", () => {
    it("prefers describer over display template when resource_details present", () => {
      const result = buildSummary(
        "protonmail.archive_email",
        { message_id: 10, folder: "INBOX" },
        null,
        "Archive Email",
        'Archive email in {{folder}}',
        { subject: "Legacy Media litigation", from: ["daniel.rose@littensipe.com"] },
      );
      expect(result).toContain("Archive email");
      expect(result).toContain("Legacy Media litigation");
      expect(result).toContain("daniel.rose@littensipe.com");
      expect(result).not.toContain('INBOX');
    });

    it("falls back to display template when resource_details absent", () => {
      const result = buildSummary(
        "protonmail.archive_email",
        { message_id: 10, folder: "INBOX" },
        null,
        "Archive Email",
        'Archive email in {{folder}}',
      );
      expect(result).toBe(`Archive email in ${q("INBOX")}`);
    });

    it("shows enriched single archive", () => {
      const result = buildSummary(
        "protonmail.archive_email",
        { message_id: 10, folder: "INBOX" },
        null,
        null,
        undefined,
        { subject: "Archive me", from: ["sender@example.com"] },
      );
      expect(result).toContain("Archive email");
      expect(result).toContain("Archive me");
      expect(result).toContain("sender@example.com");
    });

    it("shows batch archive count without per-email subjects", () => {
      const result = buildSummary(
        "protonmail.archive_email",
        { message_ids: [10, 11], folder: "INBOX" },
        null,
        null,
        undefined,
        {
          messages: {
            "10": { subject: "First", from: ["a@example.com"], to: [], date: "2026-01-01T00:00:00Z" },
            "11": { subject: "Second", from: ["b@example.com"], to: [], date: "2026-01-02T00:00:00Z" },
          },
        },
      );
      expect(result).toContain("Archive");
      expect(result).toContain("2");
      expect(result).toContain("emails");
      // Per-email subjects are rendered by ProtonBatchEmailsPreview's
      // collapsible rows, not crammed into the one-line summary.
      expect(result).not.toContain("First");
      expect(result).not.toContain("Second");
    });

    it("falls back when enrichment is absent", () => {
      const result = buildSummary(
        "protonmail.archive_email",
        { message_id: 10, folder: "INBOX" },
        null,
        "Archive Email",
      );
      expect(result).toContain("Archive Email");
      expect(result).toContain("10");
    });
  });

  describe("protonmail.reply_email", () => {
    it("shows in-reply-to subject and sender from resourceDetails", () => {
      const result = buildSummary(
        "protonmail.reply_email",
        { in_reply_to_message_id: 7, body: "Sure" },
        null,
        null,
        undefined,
        {
          in_reply_to: {
            subject: "Question",
            from: ["asker@example.com"],
            to: ["me@proton.me"],
            date: "2026-04-01T08:00:00Z",
          },
        },
      );
      expect(result).toContain("Reply to");
      expect(result).toContain("Question");
      expect(result).toContain("asker@example.com");
    });
  });

  describe("protonmail.mark_read", () => {
    it("shows enriched subject", () => {
      const result = buildSummary(
        "protonmail.mark_read",
        { message_id: 10, folder: "INBOX" },
        null,
        null,
        undefined,
        { subject: "Unread notice", from: ["alerts@example.com"] },
      );
      expect(result).toContain("Mark as read");
      expect(result).toContain("Unread notice");
    });
  });

  describe("protonmail.move_to_folder", () => {
    it("shows target folder", () => {
      const result = buildSummary(
        "protonmail.move_to_folder",
        { message_id: 10, folder: "INBOX", target_folder: "Archive" },
        null,
        null,
        undefined,
        { subject: "Move me" },
      );
      expect(result).toContain("Move email");
      expect(result).toContain("Move me");
      expect(result).toContain("Archive");
    });
  });

  describe("protonmail.list_folders", () => {
    it("shows list folders label", () => {
      const result = buildSummary(
        "protonmail.list_folders",
        {},
        null,
        "List Folders",
      );
      expect(result).toBe("List mailbox folders");
    });
  });

  // ── Existing describers still work ──────────────────────────────

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
        required: ["target", "region"],
        properties: {
          target: { type: "string", description: "Deploy target env", "x-ui": { label: "Target" } },
          region: { type: "string", description: "Cloud region", "x-ui": { label: "Region" } },
          count: { type: "integer", description: "Instance count", "x-ui": { label: "Count" } },
          dry_run: { type: "boolean", description: "Dry run mode", "x-ui": { label: "Dry run" } },
        },
      };
      const result = buildSummary(
        "test.action",
        { target: "prod", region: "us-east", count: 3, dry_run: true },
        schema,
        null,
      );
      // Uses x-ui label, not description, for display labels (#862)
      expect(result).toContain("Target");
      expect(result).toContain("Region");
      expect(result).toContain("Count");
      expect(result).not.toContain("Dry run");
    });

    it("falls back to humanized key when no x-ui label", () => {
      const schema: ParametersSchema = {
        type: "object",
        required: ["channel_id"],
        properties: {
          channel_id: {
            type: "string",
            description: "Channel ID: C… (channel), D… (DM), or G… (group DM)",
          },
        },
      };
      const result = buildSummary(
        "test.action",
        { channel_id: "C123" },
        schema,
        "Read Messages",
      );
      // Should use humanized key "Channel id", NOT the verbose description (#862)
      expect(result).toContain("Channel id");
      expect(result).not.toContain("Channel ID: C");
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

  describe("email details unavailable note", () => {
    it("shows the note for a Proton Mail email action without enrichment", () => {
      render(
        <ActionPreviewSummary
          actionType="protonmail.archive_email"
          parameters={{ message_ids: [231, 232], folder: "INBOX" }}
          schema={null}
          actionName="Archive Email"
          displayTemplate="Archive email in {{folder}}"
        />,
      );

      expect(
        screen.getByTestId("email-details-unavailable").textContent,
      ).toBe("Email details unavailable");
    });

    it("hides the note when enrichment details are present", () => {
      render(
        <ActionPreviewSummary
          actionType="protonmail.archive_email"
          parameters={{ message_ids: [231, 232], folder: "INBOX" }}
          schema={null}
          actionName="Archive Email"
          resourceDetails={{
            messages: {
              "231": { subject: "Hello", from: ["a@example.com"] },
            },
          }}
        />,
      );

      expect(
        screen.queryByTestId("email-details-unavailable"),
      ).toBeNull();
    });

    it("hides the note for reply actions enriched with in_reply_to", () => {
      render(
        <ActionPreviewSummary
          actionType="protonmail.reply_email"
          parameters={{ in_reply_to_message_id: 10, body: "Thanks!" }}
          schema={null}
          actionName="Reply to Email"
          resourceDetails={{
            in_reply_to: { subject: "Hello", from: ["a@example.com"] },
          }}
        />,
      );

      expect(
        screen.queryByTestId("email-details-unavailable"),
      ).toBeNull();
    });

    it("does not show the note for non-email actions without details", () => {
      render(
        <ActionPreviewSummary
          actionType="github.create_issue"
          parameters={{ owner: "acme", repo: "web", title: "Bug" }}
          schema={null}
          actionName="Create Issue"
        />,
      );

      expect(
        screen.queryByTestId("email-details-unavailable"),
      ).toBeNull();
    });
  });
});
