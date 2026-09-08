import { lookupResolvedResource, overlayResolvedNamesOnParams, safeHttpsHref } from "./resourceDisplay";

describe("lookupResolvedResource", () => {
  it("prefers the resources map", () => {
    expect(
      lookupResolvedResource("spreadsheet_id", "s123", {
        title: "Old",
        resources: {
          spreadsheet_id: {
            s123: {
              name: "Budget 2026",
              url: "https://docs.google.com/spreadsheets/d/s123",
            },
          },
        },
      }),
    ).toEqual({
      name: "Budget 2026",
      url: "https://docs.google.com/spreadsheets/d/s123",
    });
  });

  it("falls back to overlay keys and aliases", () => {
    expect(
      lookupResolvedResource("folder_id", "abc", {
        folder_name: "Finance Shared Drive",
        folder_url: "https://drive.google.com/drive/folders/abc",
      }),
    ).toEqual({
      name: "Finance Shared Drive",
      url: "https://drive.google.com/drive/folders/abc",
    });
    expect(
      lookupResolvedResource("spreadsheet_id", "s123", { title: "Budget 2026" }),
    ).toEqual({ name: "Budget 2026" });
  });

  it("ignores non-https URLs", () => {
    expect(
      lookupResolvedResource("file_id", "f1", {
        file_name: "Report",
        file_url: "javascript:alert(1)",
      }),
    ).toEqual({ name: "Report" });
  });
});

describe("overlayResolvedNamesOnParams", () => {
  it("replaces opaque IDs with names and keeps resource_details keys", () => {
    const got = overlayResolvedNamesOnParams(
      { spreadsheet_id: "s123", range: "A1:B2" },
      { spreadsheet_name: "Budget 2026", title: "Budget 2026" },
    );
    expect(got.spreadsheet_id).toBe("Budget 2026");
    expect(got.range).toBe("A1:B2");
    expect(got.title).toBe("Budget 2026");
  });
});

describe("safeHttpsHref", () => {
  it("allows https and rejects everything else", () => {
    expect(safeHttpsHref("https://example.com/a")).toBe("https://example.com/a");
    expect(safeHttpsHref("http://example.com/a")).toBeUndefined();
    expect(safeHttpsHref("javascript:alert(1)")).toBeUndefined();
  });
});
