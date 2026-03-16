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
    <div className="bg-slate-800 dark:bg-slate-900 overflow-hidden rounded-xl p-4">
      {/* Header icon */}
      <div className="mb-3 flex items-center gap-2">
        <div className="flex size-8 items-center justify-center rounded-lg bg-blue-500/20">
          <Mail className="size-4 text-blue-400" aria-hidden="true" />
        </div>
      </div>

      {/* Message details */}
      <div className="space-y-2">
        {to && (
          <div className="flex items-center gap-2">
            <span className="text-xs font-medium text-slate-400">To</span>
            <span className="truncate text-sm font-medium text-white">
              {to}
            </span>
          </div>
        )}
        {subject && (
          <p className="truncate text-base font-semibold text-white">
            {subject}
          </p>
        )}
        {body && (
          <p className="line-clamp-2 text-sm leading-relaxed text-slate-300">
            {truncate(body, 200)}
          </p>
        )}
      </div>
    </div>
  );
}
