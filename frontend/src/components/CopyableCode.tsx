import { useState } from "react";
import { Copy, Check } from "lucide-react";
import { toast } from "sonner";
import { cn } from "@/lib/utils";

interface CopyableCodeProps {
  code: string;
  className?: string;
}

/**
 * Inline button that displays a code string with a click-to-copy action.
 * Styled with monospace font and a copy/check icon. Pass `className` to
 * override the default background/padding for different visual contexts
 * (e.g. banner vs table row).
 */
export function CopyableCode({ code, className }: CopyableCodeProps) {
  const [copied, setCopied] = useState(false);

  function handleCopy(e: React.MouseEvent) {
    e.stopPropagation();
    navigator.clipboard.writeText(code).then(
      () => {
        setCopied(true);
        toast.success("Confirmation code copied");
        setTimeout(() => setCopied(false), 2000);
      },
      () => toast.error("Failed to copy"),
    );
  }

  return (
    <button
      onClick={handleCopy}
      className={cn(
        "inline-flex items-center gap-1.5 rounded-md font-mono text-xs font-semibold",
        className ??
          "bg-muted hover:bg-muted/80 px-2 py-1 transition-colors",
      )}
      title="Click to copy confirmation code"
      aria-label="Copy confirmation code"
    >
      {code}
      {copied ? (
        <Check className="size-3 text-green-600 dark:text-green-400" />
      ) : (
        <Copy className="size-3 opacity-60" />
      )}
    </button>
  );
}
