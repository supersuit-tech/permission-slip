import { Link } from "react-router-dom";
import { cn } from "@/lib/utils";

interface FooterProps {
  className?: string;
}

const linkClass = "hover:text-foreground transition-colors";

/** Shared site footer for auth and marketing-style layouts. */
export function Footer({ className }: FooterProps) {
  return (
    <footer className={cn("text-xs text-muted-foreground", className)}>
      <div className="flex flex-wrap gap-x-4 gap-y-2">
        <Link to="/support" className={linkClass}>
          Support
        </Link>
      </div>
    </footer>
  );
}
