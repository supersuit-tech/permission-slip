import { Button } from "@/components/ui/button";

interface ResendButtonProps {
  isResending: boolean;
  error: string | null;
  success: boolean;
  onResend: () => void;
  /** Label used for the button text and aria-label (e.g. "Resend code", "Resend email"). */
  label: string;
  /** Success message shown after a successful resend (e.g. "Code resent.", "Email resent."). */
  successMessage: string;
}

export function ResendButton({
  isResending,
  error,
  success,
  onResend,
  label,
  successMessage,
}: ResendButtonProps) {
  return (
    <div className="mt-3 flex flex-col items-start gap-1">
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={onResend}
        disabled={isResending}
        aria-label={isResending ? "Resending…" : label}
        className="opacity-70"
      >
        {isResending ? "Resending…" : label}
      </Button>
      {error && (
        <p role="alert" className="text-xs text-destructive">{error}</p>
      )}
      {success && (
        <p role="status" className="text-xs text-muted-foreground">{successMessage}</p>
      )}
    </div>
  );
}
