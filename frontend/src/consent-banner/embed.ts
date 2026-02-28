/**
 * Embeddable consent banner for permissionslip.dev and similar hosts.
 *
 * Drop this script onto any page (e.g. www.permissionslip.dev) to show the
 * same GDPR-compliant cookie consent banner used by app.permissionslip.dev.
 * Consent state is shared across subdomains of the current site via a cookie
 * whose domain is derived from the embedding page's hostname (e.g.
 * `.permissionslip.dev` when used on permissionslip.dev).
 *
 * Usage:
 *   <script src="https://app.permissionslip.dev/consent-banner.js"></script>
 *
 * The banner auto-renders at the bottom of the viewport when no consent
 * decision has been recorded. It removes itself after the user chooses.
 *
 * No dependencies — pure vanilla JS + inline styles.
 */

import {
  clearConsent,
  getStoredConsent,
  persistConsent,
  type ConsentStatus,
} from "../lib/consent-cookie";

/** Detect dark mode preference. */
function prefersDark(): boolean {
  return window.matchMedia?.("(prefers-color-scheme: dark)").matches ?? false;
}

function createBanner(): HTMLElement | null {
  // Already consented — nothing to render.
  if (getStoredConsent() !== null) return null;

  const dark = prefersDark();

  const banner = document.createElement("div");
  banner.setAttribute("role", "region");
  banner.setAttribute("aria-label", "Cookie consent");
  banner.id = "ps-consent-banner";
  banner.style.cssText = [
    "position:fixed",
    "inset-inline:0",
    "bottom:0",
    "z-index:9999",
    `border-top:1px solid ${dark ? "#374151" : "#e5e7eb"}`,
    `background:${dark ? "#1f2937" : "#fff"}`,
    "padding:16px 24px",
    `box-shadow:0 -4px 12px ${dark ? "rgba(0,0,0,.3)" : "rgba(0,0,0,.08)"}`,
    "font-family:system-ui,-apple-system,sans-serif",
    "font-size:14px",
    `color:${dark ? "#f3f4f6" : "#1f2937"}`,
  ].join(";");

  const inner = document.createElement("div");
  inner.style.cssText = [
    "max-width:1200px",
    "margin:0 auto",
    "display:flex",
    "flex-wrap:wrap",
    "align-items:center",
    "justify-content:space-between",
    "gap:16px",
  ].join(";");

  const text = document.createElement("p");
  text.style.cssText = "margin:0;line-height:1.5;flex:1 1 300px";
  // Build the paragraph using DOM API rather than innerHTML to eliminate any
  // risk of XSS if this template is ever modified to include dynamic values.
  text.appendChild(
    document.createTextNode(
      "We use cookies to analyze site usage and improve your experience. You can accept or reject non-essential cookies. See our ",
    ),
  );
  const policyLink = document.createElement("a");
  policyLink.href = "https://app.permissionslip.dev/policy/privacy";
  policyLink.style.cssText =
    "text-decoration:underline;text-underline-offset:4px;color:inherit";
  policyLink.textContent = "Privacy Policy";
  text.appendChild(policyLink);
  text.appendChild(document.createTextNode(" for details."));

  const buttons = document.createElement("div");
  buttons.style.cssText = "display:flex;gap:12px;flex-shrink:0";

  function makeButton(label: string, primary: boolean): HTMLButtonElement {
    const btn = document.createElement("button");
    btn.type = "button";
    btn.textContent = label;
    btn.style.cssText = [
      "cursor:pointer",
      "border-radius:6px",
      "padding:6px 16px",
      "font-size:14px",
      "font-weight:500",
      "line-height:1.5",
      "transition:background .15s,border-color .15s",
      primary
        ? `background:${dark ? "#f3f4f6" : "#18181b"};color:${dark ? "#18181b" : "#fff"};border:1px solid ${dark ? "#f3f4f6" : "#18181b"}`
        : `background:${dark ? "#374151" : "#fff"};color:${dark ? "#f3f4f6" : "#18181b"};border:1px solid ${dark ? "#4b5563" : "#d1d5db"}`,
    ].join(";");
    return btn;
  }

  const rejectBtn = makeButton("Reject All", false);
  const acceptBtn = makeButton("Accept All", true);

  function dismiss(choice: ConsentStatus) {
    persistConsent(choice);
    banner.remove();
  }

  rejectBtn.addEventListener("click", () => dismiss("rejected"));
  acceptBtn.addEventListener("click", () => dismiss("accepted"));

  buttons.appendChild(rejectBtn);
  buttons.appendChild(acceptBtn);
  inner.appendChild(text);
  inner.appendChild(buttons);
  banner.appendChild(inner);

  return banner;
}

function mount() {
  const banner = createBanner();
  if (banner) document.body.appendChild(banner);
}

// Auto-mount when the DOM is ready.
if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", mount);
} else {
  mount();
}

// Expose a minimal API for programmatic control.
// Cast through unknown because Window doesn't have an index signature.
(window as unknown as Record<string, unknown>).__psConsent = {
  getConsent: getStoredConsent,
  accept: () => {
    persistConsent("accepted");
    document.getElementById("ps-consent-banner")?.remove();
  },
  reject: () => {
    persistConsent("rejected");
    document.getElementById("ps-consent-banner")?.remove();
  },
  /** Clear stored consent and re-show the banner (e.g. from a "Manage Cookies" link). */
  reset: () => {
    clearConsent();
    document.getElementById("ps-consent-banner")?.remove();
    mount();
  },
};
