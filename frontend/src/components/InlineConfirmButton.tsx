import { useState } from "react";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";

interface InlineConfirmButtonProps {
  /** The trigger element shown in the default (non-confirming) state. */
  children: React.ReactNode;
  /** Label for the destructive confirm button (e.g. "Delete", "Remove"). */
  confirmLabel: string;
  /** Whether the async action is in progress. Disables buttons during processing. */
  isProcessing: boolean;
  /** Called when the user confirms the destructive action. */
  onConfirm: () => void;
}

/**
 * Two-step inline confirmation for destructive actions.
 *
 * Renders the children (e.g. a trash icon button) by default. On click,
 * swaps to a confirm/cancel button pair. Pressing confirm fires the action;
 * pressing cancel returns to the default state.
 */
export function InlineConfirmButton({
  children,
  confirmLabel,
  isProcessing,
  onConfirm,
}: InlineConfirmButtonProps) {
  const [confirming, setConfirming] = useState(false);

  if (confirming) {
    return (
      <div className="flex items-center gap-1">
        <Button
          variant="destructive"
          size="sm"
          onClick={() => {
            onConfirm();
            setConfirming(false);
          }}
          disabled={isProcessing}
        >
          {isProcessing ? (
            <Loader2 className="size-4 animate-spin" />
          ) : (
            confirmLabel
          )}
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setConfirming(false)}
          disabled={isProcessing}
        >
          Cancel
        </Button>
      </div>
    );
  }

  return (
    <div onClick={() => setConfirming(true)} role="presentation">
      {children}
    </div>
  );
}
