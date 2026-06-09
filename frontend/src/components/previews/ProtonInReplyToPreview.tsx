import { cn } from "@/lib/utils";

export const PROTON_REPLY_ACTION_TYPE = "protonmail.reply_email";

export interface ProtonInReplyToMetadata {
  subject: string;
  from: string[];
  to: string[];
  date: string;
}

function stringArray(v: unknown): string[] {
  if (!Array.isArray(v)) return [];
  return v.filter((x): x is string => typeof x === "string");
}

/**
 * Parses approval-safe `in_reply_to` metadata from resource_details.
 */
export function parseProtonInReplyTo(
  resourceDetails: Record<string, unknown> | null | undefined,
): ProtonInReplyToMetadata | null {
  if (!resourceDetails) return null;
  const raw = resourceDetails.in_reply_to;
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return null;
  const meta = raw as Record<string, unknown>;
  const subject = typeof meta.subject === "string" ? meta.subject : "";
  const from = stringArray(meta.from);
  const to = stringArray(meta.to);
  const date = typeof meta.date === "string" ? meta.date : "";
  if (!subject && from.length === 0 && to.length === 0 && !date) {
    return null;
  }
  return { subject, from, to, date };
}

function formatList(values: string[]): string | null {
  if (values.length === 0) return null;
  return values.join(", ");
}

function formatDate(iso: string): string {
  if (!iso) return "";
  try {
    const date = new Date(iso);
    if (isNaN(date.getTime())) return iso;
    return date.toLocaleString(undefined, {
      month: "short",
      day: "numeric",
      year: "numeric",
      hour: "numeric",
      minute: "2-digit",
    });
  } catch {
    return iso;
  }
}

export interface ProtonInReplyToPreviewProps {
  metadata: ProtonInReplyToMetadata | null;
  className?: string;
}

/**
 * Metadata-only preview of the email being replied to (no body or attachments).
 */
export function ProtonInReplyToPreview({
  metadata,
  className,
}: ProtonInReplyToPreviewProps) {
  if (!metadata) {
    return (
      <div
        className={cn(
          "rounded-xl border border-dashed border-border bg-muted/20 p-4 text-sm text-muted-foreground",
          className,
        )}
      >
        No source email details were included with this reply request.
      </div>
    );
  }

  const fromLine = formatList(metadata.from);
  const toLine = formatList(metadata.to);
  const dateLine = formatDate(metadata.date);

  return (
    <div className={cn("space-y-2", className)}>
      <p className="text-muted-foreground text-xs font-medium uppercase tracking-wide">
        In reply to
      </p>
      <div className="rounded-lg border border-border/80 bg-muted/20 p-3 text-sm">
        <p className="font-semibold leading-snug">
          {metadata.subject || "(No subject)"}
        </p>
        {fromLine && (
          <p className="text-muted-foreground mt-2 text-xs">
            <span className="font-medium text-foreground">From: </span>
            {fromLine}
          </p>
        )}
        {toLine && (
          <p className="text-muted-foreground text-xs">
            <span className="font-medium text-foreground">To: </span>
            {toLine}
          </p>
        )}
        {dateLine && (
          <p className="text-muted-foreground text-xs">
            <time dateTime={metadata.date}>{dateLine}</time>
          </p>
        )}
      </div>
    </div>
  );
}
