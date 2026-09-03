import { describe, it, expect } from "vitest";
import { resolvedResourceDisplayValue } from "../resourceParameterDisplay";

describe("resolvedResourceDisplayValue", () => {
  it("overlays folder_id with folder_name", () => {
    expect(
      resolvedResourceDisplayValue("folder_id", "0AKbIIKZ8knmBUk9PVA", {
        folder_name: "Finance Shared Drive",
      }),
    ).toBe("Finance Shared Drive (0AKbIIKZ8knmBUk9PVA)");
  });

  it("overlays a Shared Drive root path label", () => {
    expect(
      resolvedResourceDisplayValue("folder_id", "0AKbllKZ8knmBUk9PVA", {
        folder_name: "Chiedo's assistant drive in the / directory",
      }),
    ).toBe("Chiedo's assistant drive in the / directory (0AKbllKZ8knmBUk9PVA)");
  });

  it("overlays a nested Shared Drive folder with the drive title", () => {
    expect(
      resolvedResourceDisplayValue("folder_id", "1Xv2Naa6LjElcSK55wb9HigrLrAaYPE0d", {
        folder_name: "2026-documents in Chiedo's assistant drive",
      }),
    ).toBe("2026-documents in Chiedo's assistant drive (1Xv2Naa6LjElcSK55wb9HigrLrAaYPE0d)");
  });

  it("overlays calendar_id with calendar_name", () => {
    expect(
      resolvedResourceDisplayValue("calendar_id", "primary", {
        calendar_name: "Work Calendar",
      }),
    ).toBe("Work Calendar (primary)");
  });

  it("overlays channel with channel_name", () => {
    expect(
      resolvedResourceDisplayValue("channel", "C0123", {
        channel_name: "#general",
      }),
    ).toBe("#general (C0123)");
  });

  it("returns null when the resolved name matches the raw id", () => {
    expect(
      resolvedResourceDisplayValue("folder_id", "Receipts", {
        folder_name: "Receipts",
      }),
    ).toBeNull();
  });

  it("returns null without resource details", () => {
    expect(resolvedResourceDisplayValue("folder_id", "abc")).toBeNull();
  });
});
