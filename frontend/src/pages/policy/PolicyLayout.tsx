import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import { ArrowLeft } from "lucide-react";

interface PolicyLayoutProps {
  title: string;
  lastUpdated?: string;
  children: ReactNode;
}

/**
 * Shared layout for public-facing policy pages (Privacy, Terms, Cookies).
 * Renders outside the auth gate so anyone can view legal documents.
 */
export function PolicyLayout({ title, lastUpdated, children }: PolicyLayoutProps) {
  return (
    <div className="min-h-screen bg-background">
      <div className="mx-auto max-w-3xl px-4 py-8 md:px-6 md:py-12">
        <nav className="mb-8">
          <Link
            to="/"
            className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
          >
            <ArrowLeft className="size-4" />
            Back to Permission Slip
          </Link>
        </nav>

        <header className="mb-8">
          <div className="flex items-center gap-2.5 mb-4">
            <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary text-sm font-bold text-primary-foreground">
              P
            </span>
            <span className="text-lg font-bold">Permission Slip</span>
          </div>
          <h1 className="text-3xl font-bold tracking-tight">{title}</h1>
          {lastUpdated && (
            <p className="mt-2 text-sm text-muted-foreground">
              Last updated: {lastUpdated}
            </p>
          )}
        </header>

        <main className="prose prose-neutral dark:prose-invert max-w-none">
          {children}
        </main>

        <footer className="mt-12 border-t pt-6">
          <div className="flex flex-wrap gap-4 text-sm text-muted-foreground">
            <Link to="/policy/privacy" className="hover:text-foreground transition-colors">
              Privacy Policy
            </Link>
            <Link to="/policy/terms" className="hover:text-foreground transition-colors">
              Terms of Service
            </Link>
            <Link to="/policy/cookies" className="hover:text-foreground transition-colors">
              Cookie Policy
            </Link>
          </div>
        </footer>
      </div>
    </div>
  );
}
