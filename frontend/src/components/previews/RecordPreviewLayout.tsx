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
    <div className="bg-slate-800 dark:bg-slate-900 overflow-hidden rounded-xl p-4">
      {/* Header icon */}
      <div className="mb-3 flex items-center gap-2">
        <div className="flex size-8 items-center justify-center rounded-lg bg-emerald-500/20">
          <FileText className="size-4 text-emerald-400" aria-hidden="true" />
        </div>
      </div>

      {/* Record details */}
      <div className="space-y-2">
        {title && (
          <p className="truncate text-base font-semibold text-white">
            {title}
          </p>
        )}
        {subtitle && (
          <p className="truncate text-sm text-slate-300">{subtitle}</p>
        )}
        {extraEntries.length > 0 && (
          <div className="mt-2 space-y-1 border-t border-slate-700 pt-2">
            {extraEntries.map((entry) => (
              <div
                key={entry.label}
                className="flex items-center justify-between gap-2"
              >
                <span className="text-xs capitalize text-slate-400">
                  {entry.label}
                </span>
                <span className="truncate text-xs text-slate-300">
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
