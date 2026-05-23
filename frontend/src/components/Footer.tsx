import { Link } from "react-router-dom";
import { cn } from "@/lib/utils";

interface FooterProps {
  className?: string;
}

const linkClass = "hover:text-foreground transition-colors";

const GIT_COMMIT_HASH = import.meta.env.VITE_GIT_COMMIT_HASH ?? "unknown";
const GIT_COMMIT_TIMESTAMP =
  import.meta.env.VITE_GIT_COMMIT_TIMESTAMP ?? "unknown";

function formatCommitTimestamp(iso: string): string {
  if (iso === "unknown") return "";
  const date = new Date(iso);
  if (isNaN(date.getTime())) return "";
  return date.toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

function buildLabel(): string {
  const sha = GIT_COMMIT_HASH.slice(0, 7);
  const ts = formatCommitTimestamp(GIT_COMMIT_TIMESTAMP);
  return ts ? `Build ${sha} · ${ts}` : `Build ${sha}`;
}

/** Shared site footer for auth and marketing-style layouts. */
export function Footer({ className }: FooterProps) {
  return (
    <footer className={cn("text-xs text-muted-foreground", className)}>
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
        <Link to="/support" className={linkClass}>
          Support
        </Link>
        <span data-testid="git-commit-hash">{buildLabel()}</span>
      </div>
    </footer>
  );
}
