import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import validation from "@/lib/validation";

interface OtpCodeInputProps {
  /** HTML id for the input element (also used for label association). */
  id: string;
  /** Visible label text above the input. */
  label: string;
  /** Current input value. */
  value: string;
  /** Called with the new value. */
  onChange: (value: string) => void;
  autoFocus?: boolean;
  disabled?: boolean;
  required?: boolean;
  /** When true, restricts input to digits only and shows a numeric keyboard. */
  numericOnly?: boolean;
  /** Maximum number of characters allowed. Defaults to validation.confirmationCode.length (6). */
  maxLength?: number;
}

/**
 * A one-time-code text input with browser autofill support.
 * Used for both email OTP and TOTP MFA flows.
 */
export function OtpCodeInput({
  id,
  label,
  value,
  onChange,
  autoFocus,
  disabled,
  required,
  numericOnly,
  maxLength = validation.confirmationCode.length,
}: OtpCodeInputProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        type="text"
        inputMode={numericOnly ? "numeric" : undefined}
        maxLength={maxLength}
        value={value}
        onChange={(e) =>
          onChange(
            numericOnly
              ? e.target.value.replace(/\D/g, "")
              : e.target.value,
          )
        }
        autoComplete="one-time-code"
        autoFocus={autoFocus}
        disabled={disabled}
        required={required}
      />
    </div>
  );
}
