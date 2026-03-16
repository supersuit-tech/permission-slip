import { Calendar } from "lucide-react";

interface EventPreviewLayoutProps {
  parameters: Record<string, unknown>;
  fields: Record<string, string>;
}

function parseDate(value: unknown): Date | null {
  if (typeof value !== "string") return null;
  const d = new Date(value);
  return isNaN(d.getTime()) ? null : d;
}

function formatTime(date: Date): string {
  return date.toLocaleTimeString(undefined, {
    hour: "numeric",
    minute: "2-digit",
  });
}

function formatDuration(startDate: Date, endDate: Date): string {
  const diffMs = endDate.getTime() - startDate.getTime();
  if (diffMs <= 0) return "";
  const totalMinutes = Math.round(diffMs / 60_000);
  if (totalMinutes < 60) return `${totalMinutes} min`;
  const hours = Math.floor(totalMinutes / 60);
  const mins = totalMinutes % 60;
  if (mins === 0) return `${hours} hour${hours > 1 ? "s" : ""}`;
  return `${hours}h ${mins}m`;
}

export function EventPreviewLayout({
  parameters,
  fields,
}: EventPreviewLayoutProps) {
  const title =
    typeof parameters[fields.title ?? ""] === "string"
      ? (parameters[fields.title ?? ""] as string)
      : null;
  const startDate = parseDate(parameters[fields.start ?? ""]);
  const endDate = parseDate(parameters[fields.end ?? ""]);

  const monthStr = startDate
    ? startDate.toLocaleDateString(undefined, { month: "short" }).toUpperCase()
    : null;
  const dayStr = startDate ? String(startDate.getDate()) : null;
  const duration =
    startDate && endDate ? formatDuration(startDate, endDate) : null;

  return (
    <div className="bg-slate-800 dark:bg-slate-900 overflow-hidden rounded-xl p-4">
      {/* Action header inside the card */}
      <div className="mb-3 flex items-center gap-2">
        <div className="flex size-8 items-center justify-center rounded-lg bg-indigo-500/20">
          <Calendar className="size-4 text-indigo-400" aria-hidden="true" />
        </div>
      </div>

      {/* Event details */}
      <div className="flex items-start gap-4">
        {/* Date block */}
        {monthStr && dayStr && (
          <div className="flex flex-col items-center rounded-lg bg-slate-700/60 px-3 py-2">
            <span className="text-[10px] font-semibold tracking-wider text-slate-400">
              {monthStr}
            </span>
            <span className="text-2xl font-bold leading-tight text-white">
              {dayStr}
            </span>
          </div>
        )}

        {/* Event info */}
        <div className="min-w-0 flex-1 space-y-1">
          {title && (
            <p className="truncate text-base font-semibold capitalize text-white">
              {title}
            </p>
          )}
          {startDate && endDate && (
            <p className="text-sm text-slate-300">
              {formatTime(startDate)} → {formatTime(endDate)}
            </p>
          )}
          {duration && (
            <p className="text-xs text-slate-400">{duration}</p>
          )}
        </div>
      </div>
    </div>
  );
}
