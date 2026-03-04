/**
 * Format a dollar amount from cents (e.g. 271 → "$2.71").
 */
export function formatCents(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`;
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
