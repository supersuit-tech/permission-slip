import { describe, it, expect } from "vitest";
import {
  tryFormatDateTime,
  formatDateTime,
  formatDateTimeRange,
  humanizeKey,
  formatHighlightValue,
  formatParameterValue,
  truncate,
} from "../formatValues";

describe("formatDateTime", () => {
  it("formats on-the-hour timestamps with full month, year, and @ separator", () => {
    const result = formatDateTime("2026-06-25T20:00:00");
    expect(result).toMatch(/June 25 2026 @ 8PM/);
  });

  it("includes minutes when not on the hour", () => {
    const result = formatDateTime("2026-06-25T20:30:00");
    expect(result).toMatch(/June 25 2026 @ 8:30PM/);
  });

  it("formats all-day date-only strings without timezone shift", () => {
    const result = formatDateTime("2026-03-15");
    expect(result).toMatch(/Mar/);
    expect(result).toMatch(/15/);
    expect(result).not.toMatch(/@/);
  });

  it("returns the input for invalid date strings", () => {
    expect(formatDateTime("not-a-date")).toBe("not-a-date");
  });
});

describe("formatDateTimeRange", () => {
  it("collapses the shared date for same-day events", () => {
    const result = formatDateTimeRange(
      "2026-06-25T20:00:00",
      "2026-06-25T21:00:00",
    );
    expect(result).toMatch(/June 25 2026 @ 8PM/);
    expect(result).toMatch(/9PM/);
    expect(result).toMatch(/\u2013/);
  });

  it("shows both full timestamps for multi-day events", () => {
    const result = formatDateTimeRange(
      "2026-06-25T20:00:00",
      "2026-06-26T21:00:00",
    );
    expect(result).toMatch(/June 25 2026 @ 8PM/);
    expect(result).toMatch(/June 26 2026 @ 9PM/);
  });
});

describe("tryFormatDateTime", () => {
  it("formats an ISO 8601 datetime string", () => {
    const result = tryFormatDateTime("2026-03-15T11:00:00-04:00");
    expect(result).toBeTruthy();
    // Should contain month and day at minimum
    expect(result).toMatch(/Mar/);
    expect(result).toMatch(/15/);
  });

  it("returns null for non-date strings", () => {
    expect(tryFormatDateTime("hello world")).toBeNull();
    expect(tryFormatDateTime("")).toBeNull();
  });

  it("returns null for non-string values", () => {
    expect(tryFormatDateTime(42)).toBeNull();
    expect(tryFormatDateTime(null)).toBeNull();
    expect(tryFormatDateTime(undefined)).toBeNull();
  });

  it("returns null for invalid date strings", () => {
    expect(tryFormatDateTime("2026-99-99T99:99:99")).toBeNull();
  });
});

describe("humanizeKey", () => {
  it("converts snake_case to readable label", () => {
    expect(humanizeKey("start_time")).toBe("Start time");
    expect(humanizeKey("spreadsheet_id")).toBe("Spreadsheet id");
  });

  it("converts camelCase to readable label", () => {
    expect(humanizeKey("calendarId")).toBe("Calendar id");
  });

  it("handles single words", () => {
    expect(humanizeKey("name")).toBe("Name");
  });
});

describe("formatHighlightValue", () => {
  it("truncates long strings", () => {
    const long = "A".repeat(100);
    const result = formatHighlightValue(long, 60);
    expect(result).toHaveLength(60);
    expect(result).toMatch(/\u2026$/);
  });

  it("returns short strings as-is", () => {
    expect(formatHighlightValue("hello")).toBe("hello");
  });

  it("redacts Slack opaque IDs embedded in free-text strings", () => {
    expect(
      formatHighlightValue("deploy in:C0AMRGKRTA4 more text"),
    ).toBe("deploy in:\u2014 more text");
  });

  it("formats numbers and booleans", () => {
    expect(formatHighlightValue(42)).toBe("42");
    expect(formatHighlightValue(true)).toBe("true");
  });

  it("formats short arrays", () => {
    expect(formatHighlightValue(["a", "b"])).toBe("a, b");
  });

  it("summarizes long arrays", () => {
    expect(formatHighlightValue(["a", "b", "c", "d"])).toBe("a, b, +2 more");
  });

  it("returns null for objects", () => {
    expect(formatHighlightValue({ key: "value" })).toBeNull();
  });

  it("returns null for empty arrays", () => {
    expect(formatHighlightValue([])).toBeNull();
  });
});

describe("formatParameterValue", () => {
  it("formats null/undefined as em dash", () => {
    expect(formatParameterValue(null)).toBe("\u2014");
    expect(formatParameterValue(undefined)).toBe("\u2014");
  });

  it("detects and formats datetime strings", () => {
    const result = formatParameterValue("2026-03-15T11:00:00-04:00");
    expect(result).toMatch(/Mar/);
    expect(result).toMatch(/15/);
  });

  it("returns non-date strings as-is", () => {
    expect(formatParameterValue("hello")).toBe("hello");
  });

  it("formats numbers and booleans", () => {
    expect(formatParameterValue(42)).toBe("42");
    expect(formatParameterValue(false)).toBe("false");
  });

  it("joins arrays", () => {
    expect(formatParameterValue(["a", "b"])).toBe("a, b");
  });
});

describe("truncate", () => {
  it("returns short strings unchanged", () => {
    expect(truncate("hi", 10)).toBe("hi");
  });

  it("truncates and appends ellipsis", () => {
    expect(truncate("hello world", 8)).toBe("hello w\u2026");
  });
});
