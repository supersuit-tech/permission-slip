import { describe, it, expect } from "vitest";
import {
  buildDataWindowConstraint,
  extractDataWindowConstraint,
  formatDataWindowConstraint,
  parseDataWindowFormState,
} from "@/lib/dataWindow";

describe("formatDataWindowConstraint", () => {
  it("formats last_days", () => {
    expect(formatDataWindowConstraint({ last_days: 30 })).toBe("last 30 days");
  });

  it("formats absolute range", () => {
    expect(
      formatDataWindowConstraint({
        starts_at: "2026-07-01T00:00:00Z",
        ends_at: "2026-08-01T00:00:00Z",
      }),
    ).toMatch(/from .+ until .+/);
  });
});

describe("parseDataWindowFormState", () => {
  it("reads last_days from constraints", () => {
    const form = parseDataWindowFormState({
      chat_id: 42,
      $data_window: { last_days: 14 },
    });
    expect(form.enabled).toBe(true);
    expect(form.mode).toBe("last_days");
    expect(form.lastDays).toBe("14");
  });

  it("reads last_days from v2 structured constraints", () => {
    const form = parseDataWindowFormState({
      $version: 2,
      match: "any",
      groups: [
        {
          match: "all",
          conditions: [
            { field: "limit", op: "matches", value: "*" },
            { field: "$data_window", op: "matches", value: { last_days: 30 } },
          ],
        },
      ],
    });
    expect(form.enabled).toBe(true);
    expect(form.lastDays).toBe("30");
  });
});

describe("extractDataWindowConstraint", () => {
  it("reads from v2 structured constraints", () => {
    expect(
      extractDataWindowConstraint({
        $version: 2,
        match: "any",
        groups: [
          {
            match: "all",
            conditions: [
              { field: "$data_window", op: "matches", value: { last_days: 7 } },
            ],
          },
        ],
      }),
    ).toEqual({ last_days: 7 });
  });
});

describe("buildDataWindowConstraint", () => {
  it("builds last_days object", () => {
    expect(
      buildDataWindowConstraint({
        enabled: true,
        mode: "last_days",
        lastDays: "7",
        startsAt: "",
        endsAt: "",
      }),
    ).toEqual({ last_days: 7 });
  });
});
