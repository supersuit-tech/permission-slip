import { Mail, Github } from "lucide-react";

export function SupportPage() {
  return (
    <div className="mx-auto max-w-2xl px-6 py-16">
      <h1 className="mb-6 font-serif text-3xl font-semibold tracking-tight">Support</h1>
      <p className="text-muted-foreground">
        Need help with Permission Slip? Choose the option that works best.
      </p>

      <div className="not-prose mt-8 grid gap-4 sm:grid-cols-2">
        <a
          href="mailto:support@supersuit.tech"
          className="flex flex-col items-center gap-3 rounded-lg border bg-card p-6 text-center transition-colors hover:bg-accent"
        >
          <Mail className="size-8 text-primary" />
          <span className="text-lg font-semibold">Email Support</span>
          <span className="text-sm text-muted-foreground">
            Send us an email at support@supersuit.tech
          </span>
        </a>

        <a
          href="https://github.com/supersuit-tech/permission-slip"
          target="_blank"
          rel="noopener noreferrer"
          className="flex flex-col items-center gap-3 rounded-lg border bg-card p-6 text-center transition-colors hover:bg-accent"
        >
          <Github className="size-8 text-primary" />
          <span className="text-lg font-semibold">GitHub</span>
          <span className="text-sm text-muted-foreground">
            Open an issue or browse discussions on GitHub
          </span>
        </a>
      </div>
    </div>
  );
}
