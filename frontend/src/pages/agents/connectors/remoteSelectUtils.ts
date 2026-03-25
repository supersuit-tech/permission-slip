/** Resolve a display label from a remote-select option row. */
export function readLabel(row: Record<string, unknown>, labelKey: string): string {
  const direct = row[labelKey];
  if (typeof direct === "string" && direct.length > 0) return direct;
  const displayLabel = row.display_label;
  if (typeof displayLabel === "string" && displayLabel.length > 0)
    return displayLabel;
  const id = row.id;
  return typeof id === "string" ? id : "";
}
