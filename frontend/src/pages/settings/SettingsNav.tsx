import { Link, useLocation } from "react-router-dom";
import {
  User,
  Shield,
  CreditCard,
  Settings,
  Link2,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";

interface SettingsNavItem {
  label: string;
  path: string;
  icon: LucideIcon;
}

const settingsNavItems: SettingsNavItem[] = [
  { label: "Profile", path: "/settings/profile", icon: User },
  { label: "Security", path: "/settings/security", icon: Shield },
  { label: "Integrations", path: "/settings/integrations", icon: Link2 },
  { label: "Billing", path: "/settings/billing", icon: CreditCard },
  { label: "Account", path: "/settings/account", icon: Settings },
];

export function SettingsNav() {
  const { pathname } = useLocation();

  return (
    <>
      {/* Desktop sidebar */}
      <nav className="hidden w-[200px] shrink-0 md:block" aria-label="Settings navigation">
        <ul className="space-y-1">
          {settingsNavItems.map((item) => {
            const isActive = pathname.startsWith(item.path);
            const Icon = item.icon;
            return (
              <li key={item.path}>
                <Link
                  to={item.path}
                  aria-current={isActive ? "page" : undefined}
                  className={cn(
                    "flex items-center gap-2.5 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                    isActive
                      ? "bg-accent text-accent-foreground"
                      : "text-muted-foreground hover:bg-accent/50 hover:text-foreground",
                  )}
                >
                  <Icon className="size-4" aria-hidden="true" />
                  {item.label}
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>

      {/* Mobile horizontal tabs */}
      <nav
        className="flex gap-1 overflow-x-auto md:hidden"
        aria-label="Settings tabs"
      >
        {settingsNavItems.map((item) => {
          const isActive = pathname.startsWith(item.path);
          const Icon = item.icon;
          return (
            <Link
              key={item.path}
              to={item.path}
              aria-current={isActive ? "page" : undefined}
              className={cn(
                "flex shrink-0 items-center gap-1.5 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                isActive
                  ? "bg-accent text-accent-foreground"
                  : "text-muted-foreground hover:bg-accent/50 hover:text-foreground",
              )}
            >
              <Icon className="size-4" aria-hidden="true" />
              {item.label}
            </Link>
          );
        })}
      </nav>
    </>
  );
}
