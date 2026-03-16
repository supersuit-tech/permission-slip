import { FileText } from "lucide-react";
import { truncate } from "@/lib/formatValues";

interface RecordPreviewLayoutProps {
  parameters: Record<string, unknown>;
  fields: Record<string, string>;
}

export function RecordPreviewLayout({
  parameters,
  fields,
}: RecordPreviewLayoutProps) {
  const title =
    typeof parameters[fields.title ?? ""] === "string"
      ? (parameters[fields.title ?? ""] as string)
      : null;
  const subtitle =
    typeof parameters[fields.subtitle ?? ""] === "string"
      ? (parameters[fields.subtitle ?? ""] as string)
      : null;

  // Gather extra fields beyond title/subtitle
  const extraEntries = Object.entries(fields)
    .filter(([role]) => role !== "title" && role !== "subtitle")
    .map(([role, paramName]) => {
      const val = parameters[paramName];
      if (val == null) return null;
      const display =
        typeof val === "string"
          ? truncate(val, 80)
          : Array.isArray(val)
            ? val.map(String).join(", ")
            : String(val);
      return { label: role, value: display };
    })
    .filter(Boolean) as { label: string; value: string }[];

  return (
    <div className="overflow-hidden rounded-xl border bg-card p-4 shadow-sm">
      {/* Header icon */}
      <div className="mb-3 flex items-center gap-2">
        <div className="flex size-8 items-center justify-center rounded-lg bg-emerald-50 ring-1 ring-emerald-200 dark:bg-emerald-950 dark:ring-emerald-800">
          <FileText className="size-4 text-emerald-600 dark:text-emerald-400" aria-hidden="true" />
        </div>
        <span className="text-muted-foreground text-xs font-medium">Record</span>
      </div>

      {/* Record details */}
      <div className="space-y-2">
        {title && (
          <p className="truncate text-base font-semibold">
            {title}
          </p>
        )}
        {subtitle && (
          <p className="text-muted-foreground truncate text-sm">{subtitle}</p>
        )}
        {extraEntries.length > 0 && (
          <div className="border-border mt-2 space-y-1 border-t pt-2">
            {extraEntries.map((entry) => (
              <div
                key={entry.label}
                className="flex items-center justify-between gap-2"
              >
                <span className="text-muted-foreground text-xs capitalize">
                  {entry.label}
                </span>
                <span className="text-muted-foreground truncate text-xs">
                  {entry.value}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
