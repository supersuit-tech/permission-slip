import { CheckCircle2, Loader2, X } from "lucide-react";

interface UpgradeSuccessBannerProps {
  onDismiss: () => void;
  upgraded: boolean;
}

export function UpgradeSuccessBanner({ onDismiss, upgraded }: UpgradeSuccessBannerProps) {
  return (
    <div role="status" className="relative rounded-lg border border-emerald-200 bg-emerald-50 p-4 dark:border-emerald-800 dark:bg-emerald-950">
      <button
        onClick={onDismiss}
        className="absolute top-3 right-3 text-emerald-600 hover:text-emerald-800 dark:text-emerald-400 dark:hover:text-emerald-200"
        aria-label="Dismiss success message"
      >
        <X className="size-4" aria-hidden="true" />
      </button>
      <div className="flex items-start gap-3 pr-6">
        {upgraded ? (
          <CheckCircle2 className="mt-0.5 size-5 shrink-0 text-emerald-600 dark:text-emerald-400" aria-hidden="true" />
        ) : (
          <Loader2 className="mt-0.5 size-5 shrink-0 animate-spin text-emerald-600 dark:text-emerald-400" aria-hidden="true" />
        )}
        <div className="space-y-1">
          <p className="text-sm font-semibold text-emerald-900 dark:text-emerald-100">
            {upgraded ? "Welcome to Pay-as-you-go!" : "Activating your upgrade…"}
          </p>
          <p className="text-sm text-emerald-700 dark:text-emerald-300">
            {upgraded
              ? "You now have access to unlimited agents, credentials, and standing approvals with 90-day audit retention."
              : "Your payment was received. We're activating your plan — this usually takes a few seconds."}
          </p>
        </div>
      </div>
    </div>
  );
}
