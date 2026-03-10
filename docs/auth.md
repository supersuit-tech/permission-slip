# Frontend Authentication

The frontend uses [Supabase Auth](https://supabase.com/docs/guides/auth) with a **passwordless email** flow. No passwords are stored or managed by this application.

The authentication method differs by environment:
- **Development**: Supabase sends a **6-digit OTP code** (via local Mailpit). The user enters the code in the UI.
- **Production**: Supabase sends a **magic link**. The user clicks the link in their email to authenticate — no code entry required.

## How it works

1. User enters their email on the login page.
2. Supabase sends either a 6-digit code (dev) or a magic link (production) to that email.
3. **Development**: User enters the code → Supabase verifies it and returns a session.
   **Production**: User clicks the magic link → Supabase authenticates via URL callback and returns a session.
4. The session (JWT) is stored in the browser by the Supabase client library and refreshed automatically.

```
                           Development                              Production
                           ───────────                              ──────────
EmailStep  ──sendOtp(email)──→  Supabase Auth  ←──sendOtp(email)──  EmailStep
                                     │
                      sends email (code or link)
                                     │
OtpStep   ──verifyOtp(email, code)─→ │ ←── user clicks link ──  CheckEmailStep
                                     │         (no code input)
                              returns session
                                     │
AuthContext  ←──onAuthStateChange────┘
(sets user, session, authStatus)
```

## Module layout

All auth code lives in `frontend/src/auth/`:

| File | Purpose |
|---|---|
| `types.ts` | `AuthStatus` and `AuthState` type definitions |
| `AuthContext.tsx` | React context provider + `useAuth` hook. Wraps the Supabase auth listener and exposes `sendOtp`, `verifyOtp`, `signOut`, and MFA functions (`verifyMfa`, `enrollMfa`, `confirmMfaEnrollment`, `unenrollMfa`, `listMfaFactors`). |
| `MfaChallengePage.tsx` | Shown at login when the user has MFA enrolled but hasn't completed the second factor. Prompts for a TOTP code. |
| `LoginPage.tsx` | Step router — switches between `EmailStep`, `OtpStep` (dev), or `CheckEmailStep` (production) based on local state and `import.meta.env.DEV`. |
| `EmailStep.tsx` | Email input form. Calls `sendOtp` and advances to the next step on success. |
| `OtpStep.tsx` | 6-digit code input form (dev only). Calls `verifyOtp`. Shows a dev-only auto-fill button (see below). |
| `CheckEmailStep.tsx` | "Check your email" screen (production only). Tells the user to click their magic link. No code input. |
| `useResend.ts` | Shared hook for resend state machine — loading flag, success/error banners, cooldown-expiry cleanup. Used by both `OtpStep` and `CheckEmailStep`. |
| `ResendButton.tsx` | Shared presentational component for the resend button with cooldown countdown, loading state, and error/success banners. |
| `useFormSubmit.ts` | Shared hook for form submission — manages error state, loading state, and safe error message conversion. Used by both `EmailStep` and `OtpStep`. |
| `AuthLayout.tsx` | Shared layout wrapper for auth pages (container + heading). |
| `errors.ts` | Maps Supabase `AuthError` codes to safe, user-facing messages. Avoids leaking internal error details. |
| `dev.ts` | `fetchOtpFromMailpit` — fetches the OTP code from the local Supabase Mailpit mail server for faster local testing. Only used in dev mode. |

The `DevOnly` component (renders children only in development) lives in `frontend/src/components/DevOnly.tsx` since it's a general-purpose utility, not auth-specific.

Tests live in `frontend/src/auth/__tests__/` and use a mocked Supabase client (`frontend/src/lib/__mocks__/supabaseClient.ts`).

## Key design decisions

**AuthContext listens to `onAuthStateChange` only.** The `INITIAL_SESSION` event fires synchronously on subscribe with the current session, so there's no need for a separate `getSession()` call. This avoids a race condition where both could resolve at different times.

**Error messages are allowlisted.** Raw Supabase errors can expose provider internals (rate-limit details, OTP configuration, etc.). The `safeErrorMessage` function in `errors.ts` maps known error codes to user-friendly strings and falls back to a generic message for anything else.

**Step components are stateless with respect to auth.** `EmailStep` and `OtpStep` receive callbacks via props rather than calling Supabase directly. This makes them easy to test in isolation with mock functions.

## Sign out

`signOut()` in `AuthContext` calls `supabase.auth.signOut({ scope: "global" })`, which revokes all sessions across every browser and device — not just the current tab. This is an intentional security choice for an app that manages third-party credentials.

On failure, the error is logged via `console.error` (for devtools / log drains) and surfaced to the user as a toast notification (via [sonner](https://sonner.emilkowal.dev/)).

## Local development with Mailpit

When running Supabase locally (`supabase start`), emails are captured by [Mailpit](http://127.0.0.1:54324/mailpit) instead of being sent for real. The `OtpStep` component renders a "Dev: Auto-fill code from Mailpit" button (in dev mode only) that fetches the latest OTP code automatically so you don't have to copy-paste from the Mailpit UI.

## MFA (Multi-Factor Authentication)

MFA via TOTP (app authenticator) is enabled in the Supabase config. Users can enroll and manage TOTP factors through the Security settings dialog (UserMenu → Security).

### How it works

1. **Enrollment**: User opens Security settings → clicks "Set Up Authenticator App" → scans QR code with their authenticator app → enters verification code to confirm.
2. **Login challenge**: After email OTP verification, `AuthContext` checks the AAL (Authenticator Assurance Level). If the user has MFA enrolled (`currentLevel: "aal1"`, `nextLevel: "aal2"`), `authStatus` is set to `"mfa_required"` and `MfaChallengePage` is shown.
3. **Verification**: User enters the 6-digit TOTP code → `verifyMfa()` calls `challengeAndVerify()` → on success, session is upgraded to AAL2 → `authStatus` becomes `"authenticated"`.

### Module layout

| File | Purpose |
|---|---|
| `auth/MfaChallengePage.tsx` | Login-time TOTP challenge (shown when `authStatus === "mfa_required"`) |
| `pages/security/MfaSettingsDialog.tsx` | Dialog for managing MFA enrollment (accessed via UserMenu → Security) |
| `pages/security/MfaEnrollmentFlow.tsx` | QR code display + verification code input for TOTP enrollment |

### Key details

- `App.tsx` checks for `"mfa_required"` immediately after authentication, before profile or onboarding checks, so MFA cannot be bypassed.
- Enrollment uses `supabase.auth.mfa.enroll()` which returns a QR code data URI. The factor isn't active until confirmed with `challengeAndVerify()`.
- Users can remove their authenticator via the Security settings dialog (`unenrollMfa()`).

### Session security

Sessions are configured in [`supabase/config.toml`](../supabase/config.toml) (`[auth.sessions]` and `[auth.email]`):
- **24-hour hard timeout** (`timebox`): Forces re-authentication regardless of activity
- **8-hour inactivity timeout**: Forces re-authentication if the user has been idle
- **60s OTP email frequency** (`max_frequency`): Prevents OTP request spam

> **Note:** Changes to these values in `config.toml` require a `supabase stop && supabase start` to take effect.
