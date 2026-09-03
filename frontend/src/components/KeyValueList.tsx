/**
 * Renders a list of key-value pairs for approval parameters and context.
 * Short values display inline; long or multiline values stack below the label.
 */
export interface KeyValueEntry {
  label: string;
  value: string;
  thumbnailSrc?: string;
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
      {entries.map(({ label, value, thumbnailSrc }, index) => {
        const isLong = isLongValue(value) || Boolean(thumbnailSrc);
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
            <div
              className={
                isLong
                  ? "space-y-2"
                  : "min-w-0 text-right"
              }
            >
              {thumbnailSrc && (
                <img
                  src={thumbnailSrc}
                  alt=""
                  className="border-border max-h-24 rounded-md border object-contain"
                />
              )}
              <span
                className={
                  isLong
                    ? "text-foreground block text-sm leading-relaxed break-words whitespace-pre-wrap"
                    : "text-foreground text-sm break-all"
                }
              >
                {value}
              </span>
            </div>
          </div>
        );
      })}
    </div>
  );
}
