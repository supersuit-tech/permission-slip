/**
 * Renders a list of key-value pairs for approval parameters and context.
 * Short values display inline; long or multiline values stack below the label.
 */
export interface KeyValueEntry {
  label: string;
  value: string;
}

interface KeyValueListProps {
  entries: KeyValueEntry[];
}

function isLongValue(value: string): boolean {
  return value.length > 40 || value.includes("\n");
}

export function KeyValueList({ entries }: KeyValueListProps) {
  return (
    <div>
      {entries.map(({ label, value }, index) => {
        const isLong = isLongValue(value);
        const isLast = index === entries.length - 1;

        return (
          <div
            key={`${label}-${index}`}
            className={
              isLong
                ? `space-y-1 py-2 ${isLast ? "" : "border-border border-b"}`
                : `flex items-start justify-between gap-3 py-2 ${isLast ? "" : "border-border border-b"}`
            }
          >
            <span className="text-muted-foreground shrink-0 text-sm font-medium">
              {label}
            </span>
            <span
              className={
                isLong
                  ? "text-foreground block text-sm leading-relaxed break-words whitespace-pre-wrap"
                  : "text-foreground min-w-0 text-right text-sm break-all"
              }
            >
              {value}
            </span>
          </div>
        );
      })}
    </div>
  );
}
