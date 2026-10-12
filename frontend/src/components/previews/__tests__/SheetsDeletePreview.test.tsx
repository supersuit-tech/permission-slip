import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { parseSheetsDeletePreview, SheetsDeletePreview } from "../SheetsDeletePreview";

describe("parseSheetsDeletePreview", () => {
  it("returns null without values", () => {
    expect(parseSheetsDeletePreview({ sheet_name: "Sheet1" })).toBeNull();
    expect(parseSheetsDeletePreview(null)).toBeNull();
  });

  it("parses values, labels, and row numbers", () => {
    const preview = parseSheetsDeletePreview({
      sheet_name: "Sheet1",
      row_label: "rows 11–12",
      values: [
        ["Alice", 30],
        ["Bob", 28],
      ],
      row_numbers: [11, 12],
      values_truncated: true,
    });
    expect(preview).toEqual({
      sheetName: "Sheet1",
      rowLabel: "rows 11–12",
      values: [
        ["Alice", 30],
        ["Bob", 28],
      ],
      rowNumbers: [11, 12],
      truncated: true,
    });
  });
});

describe("SheetsDeletePreview", () => {
  it("renders tab name, row numbers, and cell values", () => {
    render(
      <SheetsDeletePreview
        preview={{
          sheetName: "Sheet1",
          rowLabel: "rows 11–12",
          values: [
            ["Alice", 30],
            ["Bob", 28],
          ],
          rowNumbers: [11, 12],
          truncated: false,
        }}
      />,
    );
    expect(screen.getByTestId("sheets-delete-preview")).toBeInTheDocument();
    expect(screen.getByText(/rows 11–12 on Sheet1/)).toBeInTheDocument();
    expect(screen.getByText("Alice")).toBeInTheDocument();
    expect(screen.getByText("Bob")).toBeInTheDocument();
    expect(screen.getByText("11")).toBeInTheDocument();
    expect(screen.getByText("12")).toBeInTheDocument();
  });
});
