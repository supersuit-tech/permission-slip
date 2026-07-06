import { useCallback, useState } from "react";
import { X } from "lucide-react";
import { toast } from "sonner";
import { cn } from "@/lib/utils";

type DeclineRequestButtonProps = {
  ariaLabel: string;
  onDecline: () => Promise<void>;
  disabled?: boolean;
  className?: string;
};

export function DeclineRequestButton({
  ariaLabel,
  onDecline,
  disabled = false,
  className,
}: DeclineRequestButtonProps) {
  const [isDeclining, setIsDeclining] = useState(false);

  const handleClick = useCallback(
    async (event: React.MouseEvent<HTMLButtonElement>) => {
      event.stopPropagation();
      if (disabled || isDeclining) return;

      setIsDeclining(true);
      try {
        await onDecline();
      } catch {
        toast.error("Failed to decline request. Please try again.");
      } finally {
        setIsDeclining(false);
      }
    },
    [disabled, isDeclining, onDecline],
  );

  return (
    <button
      type="button"
      aria-label={ariaLabel}
      disabled={disabled || isDeclining}
      onClick={handleClick}
      className={cn(
        "inline-flex size-7 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50",
        className,
      )}
    >
      <X className="size-4" aria-hidden="true" />
    </button>
  );
}
