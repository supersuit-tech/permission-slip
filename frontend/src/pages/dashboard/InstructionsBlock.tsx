import { useState, useRef, useEffect } from "react";
import { Copy, Check } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";

interface InstructionsBlockProps {
  instructions: string;
  buttonLabel?: string;
}

/**
 * Scrollable read-only block of agent instructions with a "Copy Instructions"
 * button. Used across the invite, verification, and post-registration dialogs.
 */
export function InstructionsBlock({
  instructions,
  buttonLabel = "Copy Instructions",
}: InstructionsBlockProps) {
  const [copied, setCopied] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, []);

  function handleCopy() {
    const onSuccess = () => {
      setCopied(true);
      toast.success("Instructions copied to clipboard");
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(() => setCopied(false), 2000);
    };

    if (navigator.clipboard) {
      navigator.clipboard.writeText(instructions).then(onSuccess, () =>
        toast.error("Failed to copy to clipboard"),
      );
      return;
    }

    // Fallback for non-secure contexts (HTTP), e.g. local network access
    try {
      const textarea = document.createElement("textarea");
      textarea.value = instructions;
      textarea.style.position = "fixed";
      textarea.style.opacity = "0";
      document.body.appendChild(textarea);
      textarea.focus();
      textarea.select();
      document.execCommand("copy");
      document.body.removeChild(textarea);
      onSuccess();
    } catch {
      toast.error("Failed to copy to clipboard");
    }
  }

  return (
    <div className="space-y-3">
      <div className="bg-muted max-h-64 overflow-y-auto rounded-lg border p-4">
        <pre className="whitespace-pre-wrap font-mono text-xs leading-relaxed">
          {instructions}
        </pre>
      </div>
      <Button onClick={handleCopy} className="w-full" variant="secondary">
        {copied ? (
          <Check className="mr-2 size-4 text-green-600" />
        ) : (
          <Copy className="mr-2 size-4" />
        )}
        {copied ? "Copied!" : buttonLabel}
      </Button>
    </div>
  );
}
