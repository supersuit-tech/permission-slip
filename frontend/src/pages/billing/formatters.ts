// Re-export shared currency formatter so existing billing imports keep working.
export { formatCents } from "@/lib/utils";

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

/** Stripe domains that are allowed for checkout and invoice redirects. */
const STRIPE_ALLOWED_HOSTS = new Set([
  "checkout.stripe.com",
  "invoice.stripe.com",
  "billing.stripe.com",
]);

/**
 * Validate that a URL is a safe Stripe URL. Only allows HTTPS URLs
 * on known Stripe domains, preventing open redirect if the backend
 * were ever compromised.
 */
export function isStripeUrl(url: string): boolean {
  try {
    const parsed = new URL(url);
    return parsed.protocol === "https:" && STRIPE_ALLOWED_HOSTS.has(parsed.hostname);
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
