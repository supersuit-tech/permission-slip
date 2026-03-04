/**
 * Format a dollar amount from cents (e.g. 271 → "$2.71").
 */
export function formatCents(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`;
}

/**
 * Validate that a URL is a safe HTTPS URL (not javascript:, data:, etc.).
 * Returns true only for https:// URLs.
 */
export function isSafeUrl(url: string): boolean {
  try {
    const parsed = new URL(url);
    return parsed.protocol === "https:";
  } catch {
    return false;
  }
}

/**
 * Format an ISO date string to a short locale string (e.g. "Mar 1, 2026").
 */
export function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}
