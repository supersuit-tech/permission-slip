import { Mail, Github } from "lucide-react";
import { PolicyLayout } from "../policy/PolicyLayout";

export function SupportPage() {
  return (
    <PolicyLayout title="Support">
      <p>
        Need help with Permission Slip? We&apos;re here for you. Choose the
        option that works best and we&apos;ll get back to you as soon as we can.
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
    </PolicyLayout>
  );
}
