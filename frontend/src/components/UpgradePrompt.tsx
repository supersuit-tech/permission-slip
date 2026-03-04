import { Link } from "react-router-dom";
import { ArrowUpRight } from "lucide-react";

interface UpgradePromptProps {
  feature: string;
}

/**
 * Inline upgrade CTA shown when a user hits a plan limit.
 * Links to the billing page where the user can upgrade.
 */
export function UpgradePrompt({ feature }: UpgradePromptProps) {
  return (
    <div className="flex items-center gap-2 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 dark:border-amber-800 dark:bg-amber-950/30">
      <p className="text-sm text-amber-800 dark:text-amber-200">
        {feature}{" "}
        <Link
          to="/billing"
          className="inline-flex items-center gap-0.5 font-medium underline"
        >
          Upgrade
          <ArrowUpRight className="size-3.5" />
        </Link>
      </p>
    </div>
  );
}
