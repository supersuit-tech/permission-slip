/**
 * Approval preview for google.sheets_delete_rows: tab name, 1-based
 * row/column label, and the current cell values that will disappear.
 */

export interface SheetsDeletePreviewData {
  sheetName: string;
  rowLabel: string;
  values: unknown[][];
  rowNumbers: number[];
  truncated: boolean;
}

function asCell(value: unknown): string {
  if (value == null) return "";
  if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  return JSON.stringify(value);
}

function asRow(value: unknown): unknown[] {
  if (Array.isArray(value)) return value;
  return [value];
}

export function parseSheetsDeletePreview(
  resourceDetails?: Record<string, unknown> | null,
): SheetsDeletePreviewData | null {
  if (!resourceDetails) return null;
  const raw = resourceDetails.values;
  if (!Array.isArray(raw) || raw.length === 0) return null;

  const values = raw.map(asRow);
  const rowNumbers = Array.isArray(resourceDetails.row_numbers)
    ? resourceDetails.row_numbers.filter((n): n is number => typeof n === "number")
    : [];

  return {
    sheetName: typeof resourceDetails.sheet_name === "string" ? resourceDetails.sheet_name : "",
    rowLabel: typeof resourceDetails.row_label === "string" ? resourceDetails.row_label : "",
    values,
    rowNumbers,
    truncated: resourceDetails.values_truncated === true,
  };
}

interface SheetsDeletePreviewProps {
  preview: SheetsDeletePreviewData;
}

export function SheetsDeletePreview({ preview }: SheetsDeletePreviewProps) {
  const colCount = preview.values.reduce((max, row) => Math.max(max, row.length), 0);
  const headingParts = ["Contents of"];
  if (preview.rowLabel) headingParts.push(preview.rowLabel);
  if (preview.sheetName) headingParts.push(`on ${preview.sheetName}`);
  headingParts.push("that will be deleted");

  return (
    <div className="overflow-hidden rounded-xl border bg-card p-4 shadow-sm" data-testid="sheets-delete-preview">
      <p className="text-muted-foreground mb-3 text-sm">{headingParts.join(" ")}</p>
      <div className="overflow-x-auto">
        <table className="w-full border-collapse text-sm">
          <tbody>
            {preview.values.map((row, i) => (
              <tr key={i} className="border-b last:border-0">
                {preview.rowNumbers[i] != null && (
                  <th
                    scope="row"
                    className="text-muted-foreground w-10 py-1 pr-2 text-left font-medium tabular-nums"
                  >
                    {preview.rowNumbers[i]}
                  </th>
                )}
                {Array.from({ length: colCount }, (_, c) => (
                  <td key={c} className="py-1 pr-3 last:pr-0">
                    {asCell(row[c])}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {preview.truncated && (
        <p className="text-muted-foreground mt-2 text-xs">Preview truncated — additional rows are not shown.</p>
      )}
    </div>
  );
}
