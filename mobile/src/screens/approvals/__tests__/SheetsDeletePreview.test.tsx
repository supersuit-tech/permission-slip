import { render, screen } from "@testing-library/react-native";
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
  it("renders row contents", () => {
    render(
      <SheetsDeletePreview
        preview={{
          sheetName: "Sheet1",
          rowLabel: "rows 11–12",
          values: [["Alice", 30]],
          rowNumbers: [11],
          truncated: false,
        }}
      />,
    );
    expect(screen.getByTestId("sheets-delete-preview")).toBeTruthy();
    expect(screen.getByText(/rows 11–12 on Sheet1/)).toBeTruthy();
    expect(screen.getByText(/Alice/)).toBeTruthy();
  });
});
