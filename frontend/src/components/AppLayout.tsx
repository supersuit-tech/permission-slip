import { Link, useLocation } from "react-router-dom";
import { LayoutDashboard, Activity } from "lucide-react";
import { cn } from "@/lib/utils";
import { useApprovals } from "@/hooks/useApprovals";
import { useCookieConsent } from "./CookieConsentContext";
import { UserMenu } from "./UserMenu";
import { PendingAgentBanners } from "./PendingAgentBanners";

export function AppLayout({ children }: { children: React.ReactNode }) {
  const { pathname } = useLocation();
  const isHome = pathname === "/";
  const isActivity = pathname === "/activity";
  const isSettings = pathname === "/settings";
  const { approvals } = useApprovals();
  const { reset: resetConsent } = useCookieConsent();
  const pendingCount = approvals.length;

  return (
    <div className="min-h-screen bg-background p-3 pb-20 md:p-5 md:pb-5">
      <nav className="mx-auto mb-6 flex max-w-[1200px] items-center rounded-xl border-b-2 bg-card px-4 py-3 shadow-sm md:mb-8 md:px-10 md:py-4">
        <Link to="/" className="flex items-center gap-2 text-base font-bold md:mr-10 md:gap-2.5 md:text-lg">
          <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-primary text-xs font-bold text-primary-foreground md:h-8 md:w-8 md:text-sm">
            P
          </span>
          <span className="hidden sm:inline">Permission Slip</span>
          <span className="sm:hidden">PS</span>
        </Link>
        <ul className="hidden list-none gap-6 md:flex">
          <li className={cn(
            "pb-1 font-medium",
            isHome
              ? "border-b-2 border-secondary text-foreground"
              : "text-muted-foreground"
          )}>
            <Link to="/" aria-current={isHome ? "page" : undefined} className="inline-flex items-center gap-1.5">
              Dashboard
              {pendingCount > 0 && (
                <span className="flex size-5 items-center justify-center rounded-full bg-destructive text-[10px] font-bold text-white">
                  {pendingCount > 9 ? "9+" : pendingCount}
                </span>
              )}
            </Link>
          </li>
          <li className={cn(
            "pb-1 font-medium",
            isActivity
              ? "border-b-2 border-secondary text-foreground"
              : "text-muted-foreground"
          )}>
            <Link to="/activity" aria-current={isActivity ? "page" : undefined}>Activity</Link>
          </li>
          <li className="font-medium text-muted-foreground">
            Users
          </li>
          <li className="font-medium text-muted-foreground">
            Roles
          </li>
          <li className={cn(
            "pb-1 font-medium",
            isSettings
              ? "border-b-2 border-secondary text-foreground"
              : "text-muted-foreground"
          )}>
            <Link to="/settings">Settings</Link>
          </li>
        </ul>
        <div className="ml-auto">
          <UserMenu />
        </div>
      </nav>
      <main className="mx-auto max-w-[1200px]">
        <PendingAgentBanners />
        {children}
      </main>

      <footer className="mx-auto mt-12 hidden max-w-[1200px] border-t pt-4 md:block">
        <div className="flex gap-4 text-xs text-muted-foreground">
          <Link to="/policy/privacy" className="hover:text-foreground transition-colors">Privacy Policy</Link>
          <Link to="/policy/terms" className="hover:text-foreground transition-colors">Terms of Service</Link>
          <Link to="/policy/cookies" className="hover:text-foreground transition-colors">Cookie Policy</Link>
          <a href="mailto:support@supersuit.tech" className="hover:text-foreground transition-colors">Support</a>
          <button type="button" onClick={resetConsent} className="hover:text-foreground transition-colors">Manage Cookies</button>
        </div>
      </footer>

      {/* Mobile bottom navigation */}
      <nav className="bg-card/95 fixed inset-x-0 bottom-0 z-40 border-t backdrop-blur-sm md:hidden">
        <div className="mx-auto flex max-w-md items-center justify-around px-4 py-2">
          <Link
            to="/"
            aria-current={isHome ? "page" : undefined}
            className={cn(
              "relative flex flex-col items-center gap-0.5 px-3 py-1.5 text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded-md",
              isHome ? "text-primary" : "text-muted-foreground"
            )}
          >
            <LayoutDashboard className="size-5" aria-hidden="true" />
            Dashboard
            {pendingCount > 0 && (
              <span className="absolute -top-0.5 right-0.5 flex size-4 items-center justify-center rounded-full bg-destructive text-[10px] font-bold text-white">
                {pendingCount > 9 ? "9+" : pendingCount}
              </span>
            )}
          </Link>
          <Link
            to="/activity"
            aria-current={isActivity ? "page" : undefined}
            className={cn(
              "flex flex-col items-center gap-0.5 px-3 py-1.5 text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded-md",
              isActivity ? "text-primary" : "text-muted-foreground"
            )}
          >
            <Activity className="size-5" aria-hidden="true" />
            Activity
          </Link>
        </div>
      </nav>
    </div>
  );
}
