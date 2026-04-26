/**
 * Dev-only utility for auto-filling OTP codes from Mailpit, the local
 * SMTP server bundled with Supabase CLI. Used by the "Dev: Auto-fill
 * code from Mailpit" button in OtpStep.
 *
 * Mailpit is proxied at /mailpit on the same origin (configured in both
 * vite.config.ts and dev-proxy.cjs → localhost:54324).
 *
 * OTP digit count comes from `@/lib/otpDigitLengths` (same source as
 * OtpStep / OtpCodeInput) so the Mailpit regex stays in sync with
 * `shared/validation.json` at build time.
 */

import { EMAIL_OTP_DIGIT_LENGTH } from "@/lib/otpDigitLengths";

/** Subset of the Mailpit v1 message-list response we rely on. */
interface MailpitMessageSummary {
  ID: string;
  To?: { Address: string }[];
}

interface MailpitMessageList {
  messages?: MailpitMessageSummary[];
}

/** Subset of the Mailpit v1 single-message response we rely on. */
interface MailpitMessage {
  Text?: string;
}

export async function fetchOtpFromMailpit(
  email: string
): Promise<string | null> {
  try {
    const mailpitUrl = `${window.location.origin}/mailpit`;

    const res = await fetch(`${mailpitUrl}/api/v1/messages?limit=10`);
    if (!res.ok) return null;

    const data: MailpitMessageList = await res.json();
    if (!data.messages || data.messages.length === 0) return null;

    // Find the most recent message addressed to this email
    const message = data.messages.find((msg) =>
      msg.To?.some((to) => to.Address === email)
    );
    if (!message) return null;

    // Fetch full message to read the body text
    const msgRes = await fetch(
      `${mailpitUrl}/api/v1/message/${encodeURIComponent(message.ID)}`
    );
    if (!msgRes.ok) return null;

    const fullMsg: MailpitMessage = await msgRes.json();

    const match = fullMsg.Text?.match(
      new RegExp(`\\b(\\d{${EMAIL_OTP_DIGIT_LENGTH}})\\b`),
    );
    return match?.[1] ?? null;
  } catch {
    return null;
  }
}
