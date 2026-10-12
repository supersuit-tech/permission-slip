import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import {
  parseSheetsDeletePreview,
  SheetsDeletePreview,
} from "../SheetsDeletePreview";

describe("parseSheetsDeletePreview", () => {
  it("returns null without values", () => {
    expect(parseSheetsDeletePreview({ sheet_name: "Sheet1" })).toBeNull();
    expect(parseSheetsDeletePreview(null)).toBeNull();
  });

  it("parses values and labels", () => {
    const preview = parseSheetsDeletePreview({
      sheet_name: "Sheet1",
      row_label: "rows 11–12",
      values: [["Alice"], ["Bob"]],
      row_numbers: [11, 12],
    });
    expect(preview?.sheetName).toBe("Sheet1");
    expect(preview?.rowNumbers).toEqual([11, 12]);
    expect(preview?.values).toHaveLength(2);
  });
});

describe("SheetsDeletePreview", () => {
  let renderer: ReactTestRenderer | undefined;

  afterEach(async () => {
    await act(async () => {
      renderer?.unmount();
    });
  });

  it("renders row contents", async () => {
    await act(async () => {
      renderer = create(
        createElement(SheetsDeletePreview, {
          preview: {
            sheetName: "Sheet1",
            rowLabel: "rows 11–12",
            values: [["Alice", 30]],
            rowNumbers: [11],
            truncated: false,
          },
        }),
      );
    });
    const node = renderer?.root.find((n) => n.props.testID === "sheets-delete-preview");
    expect(node).toBeTruthy();
    const texts = renderer?.root.findAllByType("Text").map((t) =>
      t.children.map(String).join(""),
    );
    expect(texts?.some((s) => s.includes("rows 11–12 on Sheet1"))).toBe(true);
    expect(texts?.some((s) => s.includes("Alice"))).toBe(true);
  });
});
