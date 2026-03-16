import type { ReactNode } from "react";
import type { LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";

interface MetadataLabelProps {
  icon?: LucideIcon;
  children: ReactNode;
  className?: string;
  title?: string;
}

/** GitHub-style metadata pill — rounded border, icon + text, compact sizing. */
export function MetadataLabel({ icon: Icon, children, className, title }: MetadataLabelProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium text-muted-foreground",
        className,
      )}
      title={title}
    >
      {Icon && <Icon className="size-3 shrink-0" aria-hidden="true" />}
      {children}
    </span>
  );
}
