import { Mail } from "lucide-react";
import { truncate } from "@/lib/formatValues";

interface MessagePreviewLayoutProps {
  parameters: Record<string, unknown>;
  fields: Record<string, string>;
}

export function MessagePreviewLayout({
  parameters,
  fields,
}: MessagePreviewLayoutProps) {
  const to =
    typeof parameters[fields.to ?? ""] === "string"
      ? (parameters[fields.to ?? ""] as string)
      : null;
  const subject =
    typeof parameters[fields.subject ?? ""] === "string"
      ? (parameters[fields.subject ?? ""] as string)
      : null;
  const body =
    typeof parameters[fields.body ?? ""] === "string"
      ? (parameters[fields.body ?? ""] as string)
      : null;

  return (
    <div className="overflow-hidden rounded-xl border bg-card p-4 shadow-sm">
      {/* Header icon */}
      <div className="mb-3 flex items-center gap-2">
        <div className="flex size-8 items-center justify-center rounded-lg bg-blue-50 ring-1 ring-blue-200 dark:bg-blue-950 dark:ring-blue-800">
          <Mail className="size-4 text-blue-600 dark:text-blue-400" aria-hidden="true" />
        </div>
        <span className="text-muted-foreground text-xs font-medium">Email</span>
      </div>

      {/* Message details */}
      <div className="space-y-2">
        {to && (
          <div className="flex items-center gap-2">
            <span className="text-muted-foreground text-xs font-medium">To</span>
            <span className="truncate text-sm font-medium">
              {to}
            </span>
          </div>
        )}
        {subject && (
          <p className="truncate text-base font-semibold">
            {subject}
          </p>
        )}
        {body && (
          <p className="text-muted-foreground line-clamp-4 text-sm leading-relaxed whitespace-pre-line">
            {truncate(body, 200)}
          </p>
        )}
      </div>
    </div>
  );
}
