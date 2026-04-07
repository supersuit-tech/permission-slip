import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import validation from "@/lib/validation";

interface OtpCodeInputProps {
  /** HTML id for the input element (also used for label association). */
  id: string;
  /** Visible label text above the input. */
  label: string;
  /** Current input value (digits only). */
  value: string;
  /** Called with the new value after non-digit characters are stripped. */
  onChange: (value: string) => void;
  autoFocus?: boolean;
  disabled?: boolean;
  required?: boolean;
}

/**
 * A 6-digit one-time-code input with numeric keyboard, digit-only filtering,
 * and browser autofill support. Used for both email OTP and TOTP MFA flows.
 *
 * Digit filtering happens on every keystroke via `.replace(/\D/g, "")` so
 * non-numeric characters can never appear in the value — even on paste.
 */
export function OtpCodeInput({
  id,
  label,
  value,
  onChange,
  autoFocus,
  disabled,
  required,
}: OtpCodeInputProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        type="text"
        inputMode="numeric"
        maxLength={validation.confirmationCode.length}
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
