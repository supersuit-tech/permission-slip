import { useId, useState } from "react";
import DOMPurify, { type Config as DOMPurifyConfig } from "dompurify";
import { Paperclip } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import type { components } from "@/api/schema";
import { cn } from "@/lib/utils";

export type EmailThread = components["schemas"]["EmailThread"];
export type EmailThreadMessage = components["schemas"]["EmailThreadMessage"];

/** Action types that include normalized `context.details.email_thread` (issue #975). */
export const EMAIL_REPLY_ACTION_TYPES = new Set([
  "google.send_email_reply",
  "microsoft.send_email_reply",
]);

/**
 * Loose runtime parse for `context.details` so we never trust JSON shape alone.
 */
export function parseEmailThreadFromDetails(details: unknown): EmailThread | null {
  if (!details || typeof details !== "object") return null;
  const d = details as Record<string, unknown>;
  const raw = d.email_thread;
  if (!raw || typeof raw !== "object") return null;
  const t = raw as Record<string, unknown>;
  if (typeof t.subject !== "string" || !Array.isArray(t.messages)) return null;
  const messages: EmailThreadMessage[] = [];
  for (const m of t.messages) {
    if (!m || typeof m !== "object") return null;
    const msg = m as Record<string, unknown>;
    if (
      typeof msg.from !== "string" ||
      !Array.isArray(msg.to) ||
      !msg.to.every((x) => typeof x === "string") ||
      !Array.isArray(msg.cc) ||
      !msg.cc.every((x) => typeof x === "string") ||
      typeof msg.date !== "string" ||
      typeof msg.body_html !== "string" ||
      typeof msg.body_text !== "string" ||
      typeof msg.snippet !== "string" ||
      typeof msg.message_id !== "string" ||
      typeof msg.truncated !== "boolean"
    ) {
      return null;
    }
    const attachmentsRaw = msg.attachments;
    let attachments: EmailThreadMessage["attachments"];
    if (attachmentsRaw !== undefined) {
      if (!Array.isArray(attachmentsRaw)) return null;
      attachments = [];
      for (const a of attachmentsRaw) {
        if (!a || typeof a !== "object") return null;
        const att = a as Record<string, unknown>;
        if (typeof att.filename !== "string" || typeof att.size_bytes !== "number") return null;
        attachments.push({ filename: att.filename, size_bytes: att.size_bytes });
      }
    }
    messages.push({
      from: msg.from,
      to: msg.to as string[],
      cc: msg.cc as string[],
      date: msg.date,
      body_html: msg.body_html,
      body_text: msg.body_text,
      snippet: msg.snippet,
      message_id: msg.message_id,
      truncated: msg.truncated,
      attachments,
    });
  }
  return { subject: t.subject, messages };
}

// DOMPurify Config type expects mutable string[] fields.
const EMAIL_HTML_SANITIZER_CONFIG: DOMPurifyConfig = {
  USE_PROFILES: { html: true },
  FORBID_TAGS: [
    "script",
    "iframe",
    "object",
    "embed",
    "base",
    "link",
    "meta",
    "form",
    "input",
    "button",
    "textarea",
    "select",
    "option",
  ],
  FORBID_ATTR: [
    "onabort",
    "onblur",
    "onchange",
    "onclick",
    "onclose",
    "oncontextmenu",
    "ondblclick",
    "onerror",
    "onfocus",
    "oninput",
    "oninvalid",
    "onkeydown",
    "onkeypress",
    "onkeyup",
    "onload",
    "onmousedown",
    "onmouseenter",
    "onmouseleave",
    "onmousemove",
    "onmouseout",
    "onmouseover",
    "onmouseup",
    "onreset",
    "onscroll",
    "onselect",
    "onsubmit",
    "onunload",
  ],
};

function sanitizeEmailHtmlFragment(html: string): string {
  return DOMPurify.sanitize(html, EMAIL_HTML_SANITIZER_CONFIG);
}

function buildIframeSrcDoc(sanitizedBody: string): string {
  return `<!DOCTYPE html><html><head><meta charset="utf-8"><style>
    body { margin: 0; padding: 8px 10px; font-family: system-ui, sans-serif; font-size: 13px; line-height: 1.45; color: #0f172a; word-wrap: break-word; overflow-wrap: anywhere; }
    img { max-width: 100%; height: auto; }
    pre { white-space: pre-wrap; word-break: break-word; }
  </style></head><body>${sanitizedBody}</body></html>`;
}

function formatListLine(label: string, values: string[]): string | null {
  if (values.length === 0) return null;
  return `${label}: ${values.join(", ")}`;
}

function MessageMeta({
  message,
  headingId,
}: {
  message: EmailThreadMessage;
  headingId: string;
}) {
  const toLine = formatListLine("To", message.to);
  const ccLine = formatListLine("Cc", message.cc);
  return (
    <div className="space-y-1 border-b border-border/60 pb-2 text-xs">
      <p id={headingId} className="font-semibold text-foreground">
        {message.from}
      </p>
      <p className="text-muted-foreground">
        <time dateTime={message.date}>{message.date}</time>
      </p>
      {toLine && <p className="text-muted-foreground">{toLine}</p>}
      {ccLine && <p className="text-muted-foreground">{ccLine}</p>}
    </div>
  );
}

function TruncationNote({ expanded, onToggle }: { expanded: boolean; onToggle: () => void }) {
  return (
    <div className="mt-2 rounded-md border border-amber-200/80 bg-amber-50/80 px-3 py-2 text-xs text-amber-950 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-100">
      {expanded ? (
        <>
          <p>
            The message body was shortened on the server (about 20 KB per field) so the approval
            screen stays responsive. The full original text is only in your mailbox.
          </p>
          <button
            type="button"
            className="mt-1 font-medium text-amber-900 underline underline-offset-2 hover:text-amber-950 dark:text-amber-200 dark:hover:text-amber-50"
            onClick={onToggle}
          >
            Show less
          </button>
        </>
      ) : (
        <>
          <p className="inline">This message was truncated server-side.</p>{" "}
          <button
            type="button"
            className="font-medium text-amber-900 underline underline-offset-2 hover:text-amber-950 dark:text-amber-200 dark:hover:text-amber-50"
            onClick={onToggle}
          >
            Show more
          </button>
        </>
      )}
    </div>
  );
}

function MessageBody({
  message,
}: {
  message: EmailThreadMessage;
}) {
  const [truncNoteOpen, setTruncNoteOpen] = useState(false);
  const html = message.body_html.trim();
  const text = message.body_text.trim();
  const useHtml = html.length > 0;
  const sanitized = useHtml ? sanitizeEmailHtmlFragment(html) : "";
  const srcDoc = useHtml ? buildIframeSrcDoc(sanitized) : "";

  return (
    <div className="space-y-2 pt-2">
      {useHtml ? (
        <iframe
          title="Email message body"
          className="w-full min-h-[160px] rounded-md border border-border bg-white dark:bg-slate-950"
          sandbox=""
          srcDoc={srcDoc}
        />
      ) : (
        <pre className="max-h-80 overflow-auto whitespace-pre-wrap break-words rounded-md border border-border bg-muted/40 p-3 text-xs leading-relaxed">
          {text || "(No body)"}
        </pre>
      )}
      {message.truncated && (
        <TruncationNote expanded={truncNoteOpen} onToggle={() => setTruncNoteOpen((v) => !v)} />
      )}
    </div>
  );
}

function AttachmentChips({
  attachments,
}: {
  attachments: NonNullable<EmailThreadMessage["attachments"]>;
}) {
  if (attachments.length === 0) return null;
  return (
    <div className="flex flex-wrap items-center gap-1.5 pt-2">
      <Paperclip className="text-muted-foreground size-3.5 shrink-0" aria-hidden="true" />
      {attachments.map((a) => (
        <Badge
          key={`${a.filename}-${a.size_bytes}`}
          variant="outline"
          className="max-w-full truncate font-normal"
        >
          {a.filename}
        </Badge>
      ))}
    </div>
  );
}

function MessageBlock({ message }: { message: EmailThreadMessage }) {
  const headingId = useId();
  return (
    <section
      aria-labelledby={headingId}
      className="rounded-lg border border-border/80 bg-muted/20 p-3"
    >
      <MessageMeta message={message} headingId={headingId} />
      <MessageBody message={message} />
      {message.attachments && message.attachments.length > 0 && (
        <AttachmentChips attachments={message.attachments} />
      )}
    </section>
  );
}

export interface EmailThreadPreviewProps {
  thread: EmailThread | null;
  className?: string;
}

/**
 * Renders normalized email thread context for approval of reply actions.
 * HTML bodies are sanitized with DOMPurify and shown in a sandboxed iframe (no scripts in approval DOM).
 */
export function EmailThreadPreview({ thread, className }: EmailThreadPreviewProps) {
  if (!thread || !Array.isArray(thread.messages) || thread.messages.length === 0) {
    return (
      <div
        className={cn(
          "rounded-xl border border-dashed border-border bg-muted/20 p-4 text-sm text-muted-foreground",
          className,
        )}
      >
        No conversation history was included with this request.
      </div>
    );
  }

  const messages = thread.messages;
  const latest = messages[messages.length - 1];
  if (!latest) {
    return (
      <div
        className={cn(
          "rounded-xl border border-dashed border-border bg-muted/20 p-4 text-sm text-muted-foreground",
          className,
        )}
      >
        No conversation history was included with this request.
      </div>
    );
  }
  const earlier = messages.length > 1 ? messages.slice(0, -1) : [];

  return (
    <div className={cn("space-y-3", className)}>
      <div>
        <h3 className="text-base font-semibold leading-snug">{thread.subject || "(No subject)"}</h3>
        <p className="text-muted-foreground mt-1 text-xs">Conversation (oldest to newest)</p>
      </div>

      {earlier.length > 0 && (
        <details className="group rounded-lg border border-border bg-card">
          <summary className="cursor-pointer list-none px-3 py-2 text-sm font-medium marker:content-none [&::-webkit-details-marker]:hidden">
            <span className="inline-flex w-full items-center justify-between gap-2">
              <span>Earlier in this thread</span>
              <span className="text-muted-foreground text-xs font-normal">
                {earlier.length} message{earlier.length === 1 ? "" : "s"}
              </span>
            </span>
          </summary>
          <div className="space-y-3 border-t border-border px-3 py-3">
            {earlier.map((m, i) => (
              <MessageBlock key={m.message_id || `earlier-${i}`} message={m} />
            ))}
          </div>
        </details>
      )}

      <div>
        <p className="text-muted-foreground mb-2 text-xs font-medium uppercase tracking-wide">
          Latest in thread
        </p>
        <MessageBlock message={latest} />
      </div>
    </div>
  );
}
