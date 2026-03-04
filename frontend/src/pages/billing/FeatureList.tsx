import { Check } from "lucide-react";

interface FeatureListProps {
  features: readonly string[] | string[];
  /** "muted" renders grey check icons (for current plan); "default" renders green (for upgrade). */
  variant?: "default" | "muted";
}

/**
 * Renders a list of plan features with check-mark icons.
 * Shared between UpgradeCTA (side-by-side comparison) and UpgradeConfirmDialog.
 */
export function FeatureList({ features, variant = "default" }: FeatureListProps) {
  const iconClass = variant === "muted" ? "text-muted-foreground" : "text-emerald-600";
  return (
    <ul className="space-y-2">
      {features.map((feature) => (
        <li key={feature} className="flex items-start gap-2 text-sm">
          <Check className={`mt-0.5 size-4 shrink-0 ${iconClass}`} aria-hidden="true" />
          <span>{feature}</span>
        </li>
      ))}
    </ul>
  );
}
