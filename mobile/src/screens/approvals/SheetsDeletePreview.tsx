import { StyleSheet, Text, View } from "react-native";
import { colors } from "../../theme/colors";

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

  return {
    sheetName: typeof resourceDetails.sheet_name === "string" ? resourceDetails.sheet_name : "",
    rowLabel: typeof resourceDetails.row_label === "string" ? resourceDetails.row_label : "",
    values: raw.map(asRow),
    rowNumbers: Array.isArray(resourceDetails.row_numbers)
      ? resourceDetails.row_numbers.filter((n): n is number => typeof n === "number")
      : [],
    truncated: resourceDetails.values_truncated === true,
  };
}

export function SheetsDeletePreview({ preview }: { preview: SheetsDeletePreviewData }) {
  const headingParts = ["Contents of"];
  if (preview.rowLabel) headingParts.push(preview.rowLabel);
  if (preview.sheetName) headingParts.push(`on ${preview.sheetName}`);
  headingParts.push("that will be deleted");

  return (
    <View style={styles.card} testID="sheets-delete-preview">
      <Text style={styles.heading}>{headingParts.join(" ")}</Text>
      {preview.values.map((row, i) => {
        const cells = row.map(asCell).join("  ·  ");
        const num = preview.rowNumbers[i];
        return (
          <View key={i} style={styles.row}>
            {num != null && <Text style={styles.rowNum}>{num}</Text>}
            <Text style={styles.cells}>{cells}</Text>
          </View>
        );
      })}
      {preview.truncated && (
        <Text style={styles.truncated}>Preview truncated — additional rows are not shown.</Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    backgroundColor: colors.white,
    borderColor: colors.gray200,
    borderRadius: 12,
    borderWidth: 1,
    padding: 16,
  },
  heading: {
    color: colors.gray500,
    fontSize: 13,
    marginBottom: 10,
  },
  row: {
    flexDirection: "row",
    alignItems: "flex-start",
    paddingVertical: 4,
  },
  rowNum: {
    color: colors.gray500,
    fontSize: 13,
    fontVariant: ["tabular-nums"],
    marginRight: 8,
    minWidth: 24,
  },
  cells: {
    color: colors.gray900,
    flex: 1,
    fontSize: 14,
  },
  truncated: {
    color: colors.gray500,
    fontSize: 12,
    marginTop: 8,
  },
});
