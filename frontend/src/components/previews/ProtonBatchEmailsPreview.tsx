import { useState } from "react";
import { cn } from "@/lib/utils";

export interface ProtonBatchEmail {
  uid: string;
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
 * Parses the batch `messages` map written by Proton Mail enrichment
 * (resource_details.messages, keyed by IMAP UID) into a sorted list.
 * Returns null unless at least two messages resolved — single-message
 * actions use flat subject/from/to/date fields instead.
 */
export function parseProtonBatchEmails(
  resourceDetails: Record<string, unknown> | null | undefined,
): ProtonBatchEmail[] | null {
  if (!resourceDetails) return null;
  const raw = resourceDetails.messages;
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return null;

  const emails: ProtonBatchEmail[] = [];
  for (const [uid, value] of Object.entries(raw as Record<string, unknown>)) {
    if (!value || typeof value !== "object" || Array.isArray(value)) continue;
    const meta = value as Record<string, unknown>;
    emails.push({
      uid,
      subject: typeof meta.subject === "string" ? meta.subject : "",
      from: stringArray(meta.from),
      to: stringArray(meta.to),
      date: typeof meta.date === "string" ? meta.date : "",
    });
  }
  if (emails.length < 2) return null;
  emails.sort((a, b) => Number(a.uid) - Number(b.uid));
  return emails;
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

interface EmailRowProps {
  email: ProtonBatchEmail;
  expanded: boolean;
  onToggle: () => void;
}

function EmailRow({ email, expanded, onToggle }: EmailRowProps) {
  const fromLine = formatList(email.from);
  const toLine = formatList(email.to);
  const dateLine = formatDate(email.date);

  return (
    <li className="border-border/80 border-b last:border-b-0">
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={expanded}
        className="hover:bg-muted/40 flex w-full items-center gap-2 px-3 py-2 text-left text-sm transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        data-testid={`batch-email-toggle-${email.uid}`}
      >
        <span
          className="text-muted-foreground shrink-0 transition-transform duration-150"
          style={{
            display: "inline-block",
            transform: expanded ? "rotate(90deg)" : "rotate(0deg)",
          }}
          aria-hidden="true"
        >
          &#9656;
        </span>
        <span className="min-w-0 flex-1 truncate font-medium">
          {email.subject || "(No subject)"}
        </span>
      </button>
      {expanded && (
        <div
          className="space-y-1 px-3 pb-3 pl-8 text-xs"
          data-testid={`batch-email-details-${email.uid}`}
        >
          {fromLine && (
            <p className="text-muted-foreground">
              <span className="text-foreground font-medium">From: </span>
              {fromLine}
            </p>
          )}
          {toLine && (
            <p className="text-muted-foreground">
              <span className="text-foreground font-medium">To: </span>
              {toLine}
            </p>
          )}
          {dateLine && (
            <p className="text-muted-foreground">
              <time dateTime={email.date}>{dateLine}</time>
            </p>
          )}
          <p className="text-muted-foreground">
            <span className="text-foreground font-medium">Message id: </span>
            {email.uid}
          </p>
        </div>
      )}
    </li>
  );
}

export interface ProtonBatchEmailsPreviewProps {
  emails: ProtonBatchEmail[];
  className?: string;
}

/**
 * Per-message preview for Proton Mail batch actions (archive, mark read,
 * move, delete, …). Each email is its own collapsible row so approvers can
 * inspect every message in the batch individually.
 */
export function ProtonBatchEmailsPreview({
  emails,
  className,
}: ProtonBatchEmailsPreviewProps) {
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  const toggle = (uid: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(uid)) {
        next.delete(uid);
      } else {
        next.add(uid);
      }
      return next;
    });
  };

  return (
    <div className={cn("space-y-2", className)} data-testid="proton-batch-emails">
      <p className="text-muted-foreground text-xs font-medium uppercase tracking-wide">
        Emails ({emails.length})
      </p>
      <ul className="border-border/80 bg-muted/20 overflow-hidden rounded-lg border">
        {emails.map((email) => (
          <EmailRow
            key={email.uid}
            email={email}
            expanded={expanded.has(email.uid)}
            onToggle={() => toggle(email.uid)}
          />
        ))}
      </ul>
    </div>
  );
}
