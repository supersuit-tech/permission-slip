# Cookie Consent Banner

GDPR-compliant cookie consent banner shared across all `permissionslip.dev` subdomains. Users only need to consent once — their choice is stored in a cross-subdomain cookie and applies to both `www.permissionslip.dev` and `app.permissionslip.dev`.

## How it works

Consent is stored in a cookie named `ps_consent` with `domain=.permissionslip.dev`. This makes it readable by all subdomains. The cookie stores one of two values: `accepted` or `rejected`.

| Property | Value |
|---|---|
| Cookie name | `ps_consent` |
| Values | `accepted` \| `rejected` |
| Domain | `.permissionslip.dev` (omitted on localhost) |
| Max-age | 1 year |
| SameSite | `Lax` |
| Secure | Yes (HTTPS only; omitted on `http://localhost` for dev) |
| Path | `/` |

## Architecture

```
frontend/
├── src/
│   ├── lib/
│   │   └── consent-cookie.ts          # Shared cookie utilities (single source of truth)
│   ├── components/
│   │   ├── CookieConsentContext.tsx    # React context + hook for in-app usage
│   │   └── CookieConsentBanner.tsx    # React banner component (uses Tailwind/shadcn)
│   └── consent-banner/
│       └── embed.ts                   # Standalone vanilla JS banner (no dependencies)
└── vite.consent-banner.config.ts      # Vite config for building embeddable script
```

### `consent-cookie.ts`

The core module. All cookie read/write/clear operations live here. Both the React context and the embeddable script import from this single source — no duplicated logic.

Exports:
- `getStoredConsent()` — read current consent from cookie (with localStorage migration)
- `persistConsent(status)` — set consent cookie
- `clearConsent()` — delete consent cookie
- `getConsentCookieDomain()` — derive the `.permissionslip.dev` domain
- `CONSENT_COOKIE_NAME` / `CONSENT_MAX_AGE` — constants

### `CookieConsentContext.tsx` + `CookieConsentBanner.tsx`

React integration for `app.permissionslip.dev`. The context provides `consent`, `accept()`, `reject()`, and `reset()`. The banner renders at the bottom of the viewport when `consent === null`.

### `embed.ts` → `consent-banner.js`

A standalone vanilla JS script (~2.7 KB, ~1.3 KB gzipped) for non-React sites. It auto-renders a consent banner using DOM API (no innerHTML) with:
- Dark mode support via `prefers-color-scheme`
- `type="button"` on all buttons (safe inside forms)
- ARIA attributes for accessibility
- A `window.__psConsent` API for programmatic control

## Usage on www.permissionslip.dev

### Option A: Script tag (simplest)

Add a single script tag to the `www` site. The banner auto-renders when needed.

```html
<script src="https://app.permissionslip.dev/consent-banner.js"></script>
```

The script exposes a `window.__psConsent` object for programmatic control:

```js
window.__psConsent.getConsent()  // "accepted" | "rejected" | null
window.__psConsent.accept()       // accept + dismiss banner
window.__psConsent.reject()       // reject + dismiss banner
```

### Option B: Import the utilities (React site)

If the `www` site is React and shares this repo (or installs it as a package), import the shared utilities directly:

```tsx
import { getStoredConsent, persistConsent, clearConsent } from '@/lib/consent-cookie';
import { CookieConsentProvider } from '@/components/CookieConsentContext';
import { CookieConsentBanner } from '@/components/CookieConsentBanner';
```

## Building

The embeddable script is built automatically as part of the standard build:

```bash
npm run build                    # builds app + consent-banner.js
npm run build:consent-banner     # builds only consent-banner.js
```

The output goes to `frontend/dist/consent-banner.js`.

## localStorage migration

Existing users who consented before this change had their preference in localStorage under the key `permission-slip-cookie-consent`. On first load after the update, `getStoredConsent()` automatically migrates the value to the new cookie and removes the localStorage entry. This is a one-time, transparent migration.

## Local development

On `http://localhost`, the `Secure` flag is omitted so cookies persist without HTTPS. The `domain` attribute is also omitted, scoping the cookie to `localhost` only (no cross-subdomain sharing needed in dev).
