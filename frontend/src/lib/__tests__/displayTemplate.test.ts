import { describe, it, expect } from "vitest";
import { renderTemplate, FORMATTERS } from "../displayTemplate";

describe("renderTemplate", () => {
  it("renders a simple template with text and values", () => {
    const parts = renderTemplate("Send email to {{to}}", { to: "alice@example.com" });
    expect(parts).toEqual([
      { kind: "text", text: "Send email to " },
      { kind: "value", text: "alice@example.com" },
    ]);
  });

  it("renders multiple placeholders", () => {
    const parts = renderTemplate("Send email to {{to}} — {{subject}}", {
      to: "bob@x.com",
      subject: "Hello",
    });
    expect(parts).toEqual([
      { kind: "text", text: "Send email to " },
      { kind: "value", text: "bob@x.com" },
      { kind: "text", text: " — " },
      { kind: "value", text: "Hello" },
    ]);
  });

  it("renders :datetime directive", () => {
    const parts = renderTemplate("Event on {{start_time:datetime}}", {
      start_time: "2026-03-15T11:00:00-04:00",
    });
    expect(parts).not.toBeNull();
    expect(parts).toHaveLength(2);
    expect(parts?.[0]).toEqual({ kind: "text", text: "Event on " });
    expect(parts?.[1]?.kind).toBe("value");
    // Should contain formatted date, not raw ISO string
    expect(parts?.[1]?.text).toMatch(/Mar/);
  });

  it("renders :count directive for arrays", () => {
    const parts = renderTemplate("with {{attendees:count}} attendees", {
      attendees: ["a@x.com", "b@x.com", "c@x.com"],
    });
    expect(parts).toEqual([
      { kind: "text", text: "with " },
      { kind: "value", text: "3" },
      { kind: "text", text: " attendees" },
    ]);
  });

  it("falls back to generic formatting when directive fails", () => {
    // :count on a non-array should fall through to formatHighlightValue
    const parts = renderTemplate("count: {{name:count}}", { name: "test" });
    expect(parts).toEqual([
      { kind: "text", text: "count: " },
      { kind: "value", text: "test" },
    ]);
  });

  it("returns null for empty template", () => {
    expect(renderTemplate("", { a: "1" })).toBeNull();
  });

  it("returns null when no placeholders resolve to values", () => {
    const parts = renderTemplate("Event on {{start_time:datetime}}", {});
    expect(parts).toBeNull();
  });

  it("handles template with no placeholders", () => {
    // A template with no {{}} placeholders and thus no values resolved
    expect(renderTemplate("Just plain text", {})).toBeNull();
  });

  it("supports custom formatters via the registry", () => {
    FORMATTERS.set("upper", (v) =>
      typeof v === "string" ? v.toUpperCase() : null,
    );
    try {
      const parts = renderTemplate("Hello {{name:upper}}", { name: "alice" });
      expect(parts).toEqual([
        { kind: "text", text: "Hello " },
        { kind: "value", text: "ALICE" },
      ]);
    } finally {
      FORMATTERS.delete("upper");
    }
  });

  it("truncates long string values", () => {
    const longVal = "A".repeat(100);
    const parts = renderTemplate("Value: {{val}}", { val: longVal });
    expect(parts).not.toBeNull();
    expect(parts?.[1]?.text.length).toBeLessThanOrEqual(60);
  });
});
