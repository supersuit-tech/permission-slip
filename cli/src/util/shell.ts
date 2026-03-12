/**
 * Shell-quoting utilities for generating safe command hints.
 *
 * These are used to produce next_step command strings that agents can safely
 * copy and run in a POSIX shell, even when values contain metacharacters.
 */

/**
 * Wraps a string in single quotes, escaping any embedded single quotes.
 * Produces output safe for POSIX /bin/sh: `'value'` or `'it'\''s ok'`.
 */
export function shellQuote(s: string): string {
  return `'${s.replace(/'/g, "'\\''")}'`;
}
