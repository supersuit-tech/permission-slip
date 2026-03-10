import { Link, useLocation } from "react-router-dom";
import { LayoutDashboard, Activity, Settings } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";
import { useApprovals } from "@/hooks/useApprovals";
import { Footer } from "./Footer";
import { UserMenu } from "./UserMenu";
import { PendingAgentBanners } from "./PendingAgentBanners";
import { BetaBanner } from "./BetaBanner";

interface NavItem {
  label: string;
  path: string;
  icon?: LucideIcon;
  badge?: number;
  disabled?: boolean;
}

function buildNavItems(pendingCount: number): NavItem[] {
  return [
    { label: "Dashboard", path: "/", icon: LayoutDashboard, badge: pendingCount },
    { label: "Activity", path: "/activity", icon: Activity },
    { label: "Settings", path: "/settings", icon: Settings },
  ];
}

export function AppLayout({ children }: { children: React.ReactNode }) {
  const { pathname } = useLocation();
  const { approvals } = useApprovals();
  const pendingCount = approvals.length;
  const navItems = buildNavItems(pendingCount);

  return (
    <div className="min-h-screen bg-background">
      <BetaBanner />
      <div className="p-3 pb-20 md:p-5 md:pb-5">
      <nav className="mx-auto mb-6 flex max-w-[1200px] items-center rounded-xl border-b-2 bg-card px-4 py-3 shadow-sm md:mb-8 md:px-10 md:py-4">
        <Link to="/" className="flex items-center gap-2 text-base font-bold md:mr-10 md:gap-2.5 md:text-lg">
          <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-primary text-xs font-bold text-primary-foreground md:h-8 md:w-8 md:text-sm">
            P
          </span>
          <span className="hidden sm:inline">Permission Slip</span>
          <span className="rounded-full bg-secondary/15 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-secondary">
            Beta
          </span>
        </Link>
        <ul className="hidden list-none gap-6 md:flex">
          {navItems.map((item) => {
            if (item.disabled) {
              return (
                <li key={item.path} className="font-medium text-muted-foreground">
                  {item.label}
                </li>
              );
            }
            const isActive = item.path === "/" ? pathname === "/" : pathname.startsWith(item.path);
            return (
              <li
                key={item.path}
                className={cn(
                  "pb-1 font-medium",
                  isActive
                    ? "border-b-2 border-secondary text-foreground"
                    : "text-muted-foreground"
                )}
              >
                <Link to={item.path} aria-current={isActive ? "page" : undefined} className="inline-flex items-center gap-1.5">
                  {item.label}
                  {item.badge != null && item.badge > 0 && (
                    <span className="flex size-5 items-center justify-center rounded-full bg-destructive text-[10px] font-bold text-white">
                      {item.badge > 9 ? "9+" : item.badge}
                    </span>
                  )}
                </Link>
              </li>
            );
          })}
        </ul>
        <div className="ml-auto">
          <UserMenu />
        </div>
      </nav>
      <main className="mx-auto max-w-[1200px]">
        <PendingAgentBanners />
        {children}
      </main>

      <Footer className="mx-auto mt-12 max-w-[1200px] border-t pt-4" />

      </div>
      {/* Mobile bottom navigation */}
      <nav className="bg-card/95 fixed inset-x-0 bottom-0 z-40 border-t backdrop-blur-sm md:hidden">
        <div className="mx-auto flex max-w-md items-center justify-around px-4 py-2">
          {navItems
            .filter((item) => item.icon && !item.disabled)
            .map((item) => {
              const isActive = item.path === "/" ? pathname === "/" : pathname.startsWith(item.path);
              const Icon = item.icon!;
              return (
                <Link
                  key={item.path}
                  to={item.path}
                  aria-current={isActive ? "page" : undefined}
                  className={cn(
                    "relative flex flex-col items-center gap-0.5 px-3 py-1.5 text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded-md",
                    isActive ? "text-primary" : "text-muted-foreground"
                  )}
                >
                  <Icon className="size-5" aria-hidden="true" />
                  {item.label}
                  {item.badge != null && item.badge > 0 && (
                    <span className="absolute -top-0.5 right-0.5 flex size-4 items-center justify-center rounded-full bg-destructive text-[10px] font-bold text-white">
                      {item.badge > 9 ? "9+" : item.badge}
                    </span>
                  )}
                </Link>
              );
            })}
        </div>
      </nav>
    </div>
  );
}
