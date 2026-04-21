import DOMPurify from "dompurify";

const SANITIZE_CONFIG = {
  ALLOWED_TAGS: ["br", "strong", "em", "s", "code", "pre", "blockquote", "a"],
  ALLOWED_ATTR: ["href", "rel", "target"],
};

/**
 * Converts Slack mrkdwn to sanitized HTML.
 *
 * Assumes the backend has already resolved <@U123> → @alice and
 * <#C123|name> → #name. Handles: *bold*, _italic_, ~strike~, `code`,
 * ```code blocks```, >blockquotes, <url|label> links, and \n → <br>.
 *
 * Processing order: code extraction → link substitution → inline
 * formatting → restore code → newlines → DOMPurify. Code spans are
 * extracted first via null-byte placeholders so that Slack link syntax
 * or mrkdwn markers inside code are never processed.
 *
 * Output must be rendered via dangerouslySetInnerHTML — DOMPurify is
 * applied as the final step.
 */
function escapeHtml(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

export function slackMrkdwnToHtml(text: string): string {
  const protected_: string[] = [];

  let html = text;

  // Extract code blocks (triple backtick) first — protect from all further transforms.
  // HTML-escape the content so angle brackets inside code render as text, not tags.
  html = html.replace(/```([\s\S]*?)```/g, (_, code: string) => {
    const idx = protected_.length;
    protected_.push(`<pre><code>${escapeHtml(code)}</code></pre>`);
    return `\x00P${idx}\x00`;
  });

  // Extract inline code — same escaping treatment
  html = html.replace(/`([^`\n]+)`/g, (_, code: string) => {
    const idx = protected_.length;
    protected_.push(`<code>${escapeHtml(code)}</code>`);
    return `\x00P${idx}\x00`;
  });

  // Slack link format: <url> and <url|label> (after code extraction)
  html = html.replace(
    /<(https?:\/\/[^|>\s]+)(?:\|([^>]*))?>/g,
    (_, url: string, label?: string) =>
      `<a href="${url}" target="_blank" rel="noopener noreferrer">${label ?? url}</a>`,
  );

  // Bold *text*
  html = html.replace(
    /\*([^*\n]+)\*/g,
    (_, t: string) => `<strong>${t}</strong>`,
  );

  // Italic _text_
  html = html.replace(/_([^_\n]+)_/g, (_, t: string) => `<em>${t}</em>`);

  // Strikethrough ~text~
  html = html.replace(/~([^~\n]+)~/g, (_, t: string) => `<s>${t}</s>`);

  // Blockquotes — lines starting with >
  html = html.replace(
    /^>(.+)$/gm,
    (_, content: string) => `<blockquote>${content}</blockquote>`,
  );

  // Restore protected code spans
  html = html.replace(
    /\x00P(\d+)\x00/g,
    (_, idx: string) => protected_[Number(idx)] ?? "",
  );

  // Newlines → <br>
  html = html.replace(/\n/g, "<br>");

  return DOMPurify.sanitize(html, SANITIZE_CONFIG);
}
