import { describe, it, expect } from "vitest";
import { resolvedResourceDisplayValue, resolvedResourceHref } from "../resourceParameterDisplay";

describe("resolvedResourceDisplayValue", () => {
  it("overlays folder_id with folder_name only", () => {
    expect(
      resolvedResourceDisplayValue("folder_id", "0AKbIIKZ8knmBUk9PVA", {
        folder_name: "Finance Shared Drive",
      }),
    ).toBe("Finance Shared Drive");
  });

  it("overlays a Shared Drive root path label", () => {
    expect(
      resolvedResourceDisplayValue("folder_id", "0AKbllKZ8knmBUk9PVA", {
        folder_name: "Chiedo's assistant drive in the / directory",
      }),
    ).toBe("Chiedo's assistant drive in the / directory");
  });

  it("overlays a nested Shared Drive folder with the drive title", () => {
    expect(
      resolvedResourceDisplayValue("folder_id", "1Xv2Naa6LjElcSK55wb9HigrLrAaYPE0d", {
        folder_name: "2026-documents in Chiedo's assistant drive",
      }),
    ).toBe("2026-documents in Chiedo's assistant drive");
  });

  it("overlays calendar_id with calendar_name", () => {
    expect(
      resolvedResourceDisplayValue("calendar_id", "primary", {
        calendar_name: "Work Calendar",
      }),
    ).toBe("Work Calendar");
  });

  it("overlays channel with channel_name", () => {
    expect(
      resolvedResourceDisplayValue("channel", "C0123", {
        channel_name: "#general",
      }),
    ).toBe("#general");
  });

  it("returns the name even when it matches the raw id", () => {
    expect(
      resolvedResourceDisplayValue("folder_id", "Receipts", {
        folder_name: "Receipts",
      }),
    ).toBe("Receipts");
  });

  it("prefers the resources map and URL", () => {
    expect(
      resolvedResourceDisplayValue("spreadsheet_id", "s123", {
        title: "Other",
        resources: {
          spreadsheet_id: {
            s123: {
              name: "Budget 2026",
              url: "https://docs.google.com/spreadsheets/d/s123",
            },
          },
        },
      }),
    ).toBe("Budget 2026");
    expect(
      resolvedResourceHref("spreadsheet_id", "s123", {
        resources: {
          spreadsheet_id: {
            s123: {
              name: "Budget 2026",
              url: "https://docs.google.com/spreadsheets/d/s123",
            },
          },
        },
      }),
    ).toBe("https://docs.google.com/spreadsheets/d/s123");
  });

  it("returns null without resource details", () => {
    expect(resolvedResourceDisplayValue("folder_id", "abc")).toBeNull();
  });
});
