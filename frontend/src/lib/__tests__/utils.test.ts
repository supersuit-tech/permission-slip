import { describe, it, expect, vi, afterEach } from "vitest";
import { formatRelativeTime } from "../utils";

describe("formatRelativeTime", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it('returns "—" for undefined', () => {
    expect(formatRelativeTime(undefined)).toBe("—");
  });

  it('returns "—" for null', () => {
    expect(formatRelativeTime(null)).toBe("—");
  });

  it('returns "—" for invalid date strings', () => {
    expect(formatRelativeTime("not-a-date")).toBe("—");
    expect(formatRelativeTime("")).toBe("—");
  });

  it('returns "Just now" for timestamps less than a minute ago', () => {
    const now = new Date("2026-02-19T12:00:00Z");
    vi.setSystemTime(now);
    expect(formatRelativeTime("2026-02-19T12:00:00Z")).toBe("Just now");
    expect(formatRelativeTime("2026-02-19T11:59:30Z")).toBe("Just now");
  });

  it("returns minutes for timestamps less than an hour ago", () => {
    vi.setSystemTime(new Date("2026-02-19T12:00:00Z"));
    expect(formatRelativeTime("2026-02-19T11:55:00Z")).toBe("5m ago");
    expect(formatRelativeTime("2026-02-19T11:31:00Z")).toBe("29m ago");
  });

  it("returns hours for timestamps less than a day ago", () => {
    vi.setSystemTime(new Date("2026-02-19T12:00:00Z"));
    expect(formatRelativeTime("2026-02-19T10:00:00Z")).toBe("2h ago");
    expect(formatRelativeTime("2026-02-18T13:00:00Z")).toBe("23h ago");
  });

  it("returns days for timestamps less than 30 days ago", () => {
    vi.setSystemTime(new Date("2026-02-19T12:00:00Z"));
    expect(formatRelativeTime("2026-02-18T10:00:00Z")).toBe("1d ago");
    expect(formatRelativeTime("2026-02-05T12:00:00Z")).toBe("14d ago");
  });

  it("returns formatted date for timestamps over 30 days ago", () => {
    vi.setSystemTime(new Date("2026-02-19T12:00:00Z"));
    const result = formatRelativeTime("2026-01-01T12:00:00Z");
    // toLocaleDateString output varies by locale, just check it's not a relative string
    expect(result).not.toContain("ago");
    expect(result).not.toBe("—");
  });
});
