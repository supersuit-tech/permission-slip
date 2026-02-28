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
    navigator.clipboard.writeText(instructions).then(
      () => {
        setCopied(true);
        toast.success("Instructions copied to clipboard");
        if (timerRef.current) clearTimeout(timerRef.current);
        timerRef.current = setTimeout(() => setCopied(false), 2000);
      },
      () => toast.error("Failed to copy to clipboard"),
    );
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
