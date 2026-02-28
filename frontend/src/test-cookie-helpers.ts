import { CONSENT_COOKIE_NAME } from "./lib/consent-cookie";

/** Set a cookie in the test environment. */
export function setCookie(name: string, value: string) {
  document.cookie = `${name}=${encodeURIComponent(value)}; path=/`;
}

/** Clear a cookie by expiring it. */
export function clearCookie(name: string) {
  document.cookie = `${name}=; path=/; max-age=0`;
}

/** Read a cookie value by name. Handles values containing '=' (e.g. base64). */
export function getCookie(name: string): string | null {
  const match = document.cookie
    .split(";")
    .map((s) => s.trim())
    .find((row) => row.startsWith(name + "="));
  if (!match) return null;
  const eqIndex = match.indexOf("=");
  const encodedValue = eqIndex >= 0 ? match.slice(eqIndex + 1) : "";
  return decodeURIComponent(encodedValue);
}

/** Clear the consent cookie — convenience for beforeEach/afterEach. */
export function clearConsentCookie() {
  clearCookie(CONSENT_COOKIE_NAME);
}
