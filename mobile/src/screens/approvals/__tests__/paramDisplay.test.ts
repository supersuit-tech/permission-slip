import { resolvedResourceDisplayValue, formatBinaryParamSummary } from "../paramDisplay";

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
});

describe("formatBinaryParamSummary", () => {
  it("summarizes a PDF instead of dumping base64", () => {
    const summary = formatBinaryParamSummary("JVBERi0xLjQK", {
      mime_type: "application/pdf",
    });
    expect(summary).toMatch(/^PDF /);
    expect(summary).not.toContain("JVBERi");
  });
});
