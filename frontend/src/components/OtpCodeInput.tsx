import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { EMAIL_OTP_DIGIT_LENGTH, TOTP_DIGIT_LENGTH } from "@/lib/otpDigitLengths";

interface OtpCodeInputProps {
  /** HTML id for the input element (also used for label association). */
  id: string;
  /** Visible label text above the input. */
  label: string;
  /** Email magic-link OTP vs authenticator TOTP — controls max digit length. */
  variant: "email" | "totp";
  /** Current input value (digits only). */
  value: string;
  /** Called with the new value after non-digit characters are stripped. */
  onChange: (value: string) => void;
  autoFocus?: boolean;
  disabled?: boolean;
  required?: boolean;
}

/**
 * One-time-code input with numeric keyboard, digit-only filtering, and browser
 * autofill support. Email OTP and TOTP use different digit lengths from
 * shared/validation.json.
 *
 * Digit filtering happens on every keystroke via `.replace(/\D/g, "")` so
 * non-numeric characters can never appear in the value — even on paste.
 */
export function OtpCodeInput({
  id,
  label,
  variant,
  value,
  onChange,
  autoFocus,
  disabled,
  required,
}: OtpCodeInputProps) {
  const maxLength =
    variant === "email" ? EMAIL_OTP_DIGIT_LENGTH : TOTP_DIGIT_LENGTH;
  return (
    <div className="space-y-2">
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        type="text"
        inputMode="numeric"
        maxLength={maxLength}
        value={value}
        onChange={(e) => onChange(e.target.value.replace(/\D/g, ""))}
        autoComplete="one-time-code"
        autoFocus={autoFocus}
        disabled={disabled}
        required={required}
      />
    </div>
  );
}
