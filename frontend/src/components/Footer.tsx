import { Link } from "react-router-dom";
import { cn } from "@/lib/utils";

interface FooterProps {
  className?: string;
}

const linkClass = "hover:text-foreground transition-colors";

function getWebVersion(): string {
  return import.meta.env.VITE_WEB_VERSION ?? "unknown";
}

/** Shared site footer for auth and marketing-style layouts. */
export function Footer({ className }: FooterProps) {
  return (
    <footer className={cn("text-xs text-muted-foreground", className)}>
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
        <Link to="/support" className={linkClass}>
          Support
        </Link>
        <span data-testid="app-version">v{getWebVersion()}</span>
      </div>
    </footer>
  );
}
