import type { LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";

interface MetadataLabelProps {
  icon?: LucideIcon;
  children: React.ReactNode;
  className?: string;
}

/** GitHub-style metadata pill — rounded border, icon + text, compact sizing. */
export function MetadataLabel({ icon: Icon, children, className }: MetadataLabelProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium text-muted-foreground",
        className,
      )}
    >
      {Icon && <Icon className="size-3 shrink-0" />}
      {children}
    </span>
  );
}
