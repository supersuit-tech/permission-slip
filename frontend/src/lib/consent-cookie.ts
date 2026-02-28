/**
 * Shared cookie utilities for the consent banner.
 *
 * Used by both the in-app React context (CookieConsentContext) and the
 * standalone embeddable banner script, ensuring identical cookie behaviour
 * across subdomains (e.g. www.permissionslip.dev and app.permissionslip.dev).
 */

export type ConsentStatus = "accepted" | "rejected";

export const CONSENT_COOKIE_NAME = "ps_consent";
/** 1 year in seconds. */
export const CONSENT_MAX_AGE = 365 * 24 * 60 * 60;

/**
 * Derive the cookie domain from the current hostname so the consent cookie is
 * shared across all subdomains (e.g. www. and app.). Uses a simple last-two-
 * labels heuristic — correct for simple TLDs like `.dev`, `.com`, `.io`.
 *
 * NOTE: This does NOT consult a public suffix list, so it would be incorrect
 * for multi-part TLDs (e.g. `.co.uk`). Since Permission Slip uses
 * `permissionslip.dev`, this is fine. If the domain ever changes to a
 * multi-part TLD, this function will need updating.
 *
 * Returns null for localhost and IP addresses (no cross-subdomain sharing).
 */
export function getConsentCookieDomain(): string | null {
  const host = window.location.hostname;
  // localhost / IP addresses — no cross-subdomain sharing needed.
  // Regex covers IPv4 (192.168.0.1) and IPv6 bracket literals ([::1]).
  if (
    host === "localhost" ||
    /^[\d.]+$/.test(host) ||
    host.startsWith("[")
  ) {
    return null;
  }
  // Strip leading subdomain(s) to get the registrable domain.
  // "app.permissionslip.dev" → ".permissionslip.dev"
  const parts = host.split(".");
  if (parts.length >= 2) {
    return "." + parts.slice(-2).join(".");
  }
  return null;
}

/** Read the raw consent cookie value. Returns null if absent or on error. */
function readRawCookie(): string | null {
  try {
    // Split on ";" and trim — document.cookie isn't guaranteed to include a
    // space after the semicolon in every browser/environment.
    const match = document.cookie
      .split(";")
      .map((s) => s.trim())
      .find((row) => row.startsWith(CONSENT_COOKIE_NAME + "="));
    if (!match) return null;
    // Split on the first "=" only — safe even if the value contains "=".
    const eqIndex = match.indexOf("=");
    return eqIndex >= 0 ? decodeURIComponent(match.slice(eqIndex + 1)) : null;
  } catch {
    // Cookie access may be blocked.
    return null;
  }
}

/**
 * Migrate consent from the old localStorage key to the shared cookie.
 * Runs once — if the cookie already exists, this is a no-op. After migrating
 * the value, the localStorage entry is removed only after verifying the cookie
 * was actually written (prevents data loss when cookies are blocked).
 */
const OLD_STORAGE_KEY = "permission-slip-cookie-consent";

function migrateFromLocalStorage(): ConsentStatus | null {
  try {
    const stored = localStorage.getItem(OLD_STORAGE_KEY);
    if (stored === "accepted" || stored === "rejected") {
      persistConsent(stored);
      // Only remove the old key if the cookie was actually written.
      // If persistConsent silently failed (e.g. cookies blocked), keeping
      // the localStorage entry prevents the user's consent from being lost.
      if (readRawCookie() === stored) {
        localStorage.removeItem(OLD_STORAGE_KEY);
      }
      return stored;
    }
  } catch {
    // localStorage may be unavailable.
  }
  return null;
}

/** Read the current consent value from the cookie, migrating from localStorage if needed. */
export function getStoredConsent(): ConsentStatus | null {
  const value = readRawCookie();
  if (value === "accepted" || value === "rejected") return value;
  // No valid cookie — check if there's an old localStorage value to migrate.
  return migrateFromLocalStorage();
}

/** Build shared cookie attributes. Secure is only set over HTTPS to avoid
 *  silent failures on http://localhost during development. */
function cookieAttrs(): string {
  const domain = getConsentCookieDomain();
  const domainAttr = domain ? `; domain=${domain}` : "";
  const secure = window.location.protocol === "https:" ? "; Secure" : "";
  return `; path=/; SameSite=Lax${secure}${domainAttr}`;
}

/** Persist a consent choice as a cross-subdomain cookie. */
export function persistConsent(status: ConsentStatus) {
  try {
    document.cookie = `${CONSENT_COOKIE_NAME}=${encodeURIComponent(status)}${cookieAttrs()}; max-age=${CONSENT_MAX_AGE}`;
  } catch {
    // Cookie access may be unavailable; consent still works for the session.
  }
}

/** Remove the consent cookie (causes the banner to reappear). */
export function clearConsent() {
  try {
    document.cookie = `${CONSENT_COOKIE_NAME}=${cookieAttrs()}; max-age=0`;
  } catch {
    // Ignore cookie errors.
  }
}
