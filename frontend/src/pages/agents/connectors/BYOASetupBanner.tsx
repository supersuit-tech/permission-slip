import { useState } from "react";
import { AlertTriangle, ExternalLink, KeyRound, LogIn } from "lucide-react";
import { Button } from "@/components/ui/button";
import { providerLabel, PROVIDER_DEV_CONSOLE_URLS } from "@/lib/oauth";
import { BYOAConfigDialog } from "@/pages/settings/BYOAConfigDialog";

interface BYOASetupBannerProps {
  providerId: string;
}

export function BYOASetupBanner({ providerId }: BYOASetupBannerProps) {
  const [configDialogOpen, setConfigDialogOpen] = useState(false);
  const label = providerLabel(providerId);
  const devConsoleUrl = PROVIDER_DEV_CONSOLE_URLS[providerId];

  return (
    <>
      <div className="mt-3 rounded-md border border-amber-200 bg-amber-50 p-4 dark:border-amber-900 dark:bg-amber-950/50">
        <div className="mb-3 flex items-center gap-2">
          <AlertTriangle className="size-4 text-amber-600 dark:text-amber-400" />
          <p className="text-sm font-semibold text-amber-800 dark:text-amber-200">
            OAuth app setup required
          </p>
        </div>
        <p className="mb-3 text-xs text-amber-700 dark:text-amber-300">
          {label} requires you to create your own OAuth app and enter the
          credentials before you can connect.
        </p>
        <ol className="space-y-3 text-sm">
          <li className="flex items-start gap-3">
            <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-amber-200 text-xs font-bold text-amber-800 dark:bg-amber-800 dark:text-amber-200">
              1
            </span>
            <div>
              <p className="font-medium text-amber-900 dark:text-amber-100">
                Create an OAuth app
              </p>
              {devConsoleUrl ? (
                <a
                  href={devConsoleUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="mt-0.5 inline-flex items-center gap-1 text-xs text-amber-700 underline hover:text-amber-900 dark:text-amber-300 dark:hover:text-amber-100"
                >
                  <ExternalLink className="size-3" />
                  Open {label} developer console
                </a>
              ) : (
                <p className="mt-0.5 text-xs text-amber-600 dark:text-amber-400">
                  Visit the {label} developer portal to create an OAuth
                  application.
                </p>
              )}
            </div>
          </li>
          <li className="flex items-start gap-3">
            <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-amber-200 text-xs font-bold text-amber-800 dark:bg-amber-800 dark:text-amber-200">
              2
            </span>
            <div>
              <p className="font-medium text-amber-900 dark:text-amber-100">
                Enter your Client ID and Client Secret
              </p>
              <Button
                variant="outline"
                size="sm"
                className="mt-1.5"
                onClick={() => setConfigDialogOpen(true)}
              >
                <KeyRound className="size-3" />
                Configure credentials
              </Button>
            </div>
          </li>
          <li className="flex items-start gap-3">
            <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-amber-200 text-xs font-bold text-amber-800 dark:bg-amber-800 dark:text-amber-200">
              3
            </span>
            <div>
              <p className="font-medium text-amber-900 dark:text-amber-100">
                Connect your account
              </p>
              <p className="mt-0.5 flex items-center gap-1 text-xs text-amber-600 dark:text-amber-400">
                <LogIn className="size-3" />
                Available after credentials are saved
              </p>
            </div>
          </li>
        </ol>
      </div>

      <BYOAConfigDialog
        open={configDialogOpen}
        onOpenChange={setConfigDialogOpen}
        provider={providerId}
        providerLabel={label}
      />
    </>
  );
}
