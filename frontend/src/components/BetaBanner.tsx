import { Link } from "react-router-dom";
import { GITHUB_REPO_URL } from "@/lib/links";

/**
 * Persistent beta banner shown at the top of every authenticated page.
 * Reminds users that the product is in early access and links to the
 * open source repository and Terms of Service for verification.
 */
export function BetaBanner() {
  return (
    <div
      role="banner"
      aria-label="Beta notice"
      className="sticky top-0 z-50 w-full border-b border-secondary/20 bg-card px-4 py-2 text-center text-xs text-foreground"
    >
      <span className="font-semibold">Beta</span>
      {" — "}
      This product is in early access. It is up to you to verify the code in our{" "}
      <a
        href={GITHUB_REPO_URL}
        target="_blank"
        rel="noopener noreferrer"
        className="underline underline-offset-2 hover:opacity-75"
      >
        open source repository
      </a>{" "}
      and{" "}
      <Link
        to="/policy/terms"
        className="underline underline-offset-2 hover:opacity-75"
      >
        review our Terms of Service
      </Link>{" "}
      before use.
    </div>
  );
}
