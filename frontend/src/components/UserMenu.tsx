import { useNavigate } from "react-router-dom";
import {
  LogOut,
  User,
  Shield,
  CreditCard,
  Moon,
  LifeBuoy,
} from "lucide-react";
import { useAuth } from "@/auth/AuthContext";
import { useProfile } from "@/hooks/useProfile";
import { useSignOut } from "@/hooks/useSignOut";
import { useTheme } from "@/components/ThemeContext";
import { Avatar } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuCheckboxItem,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

export function UserMenu() {
  const { user } = useAuth();
  const { profile } = useProfile();
  const { theme, toggleTheme } = useTheme();
  const handleSignOut = useSignOut();
  const navigate = useNavigate();

  const email = user?.email ?? "unknown";
  const username = profile?.username;

  return (
      <DropdownMenu>
        <DropdownMenuTrigger
          className="flex cursor-pointer select-none items-center gap-2 rounded-full border-none bg-transparent p-0 outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
          aria-label="User menu"
        >
          <Avatar as="span" name={username} email={email} />
          {username && (
            <span className="hidden text-sm font-medium md:inline">
              {username}
            </span>
          )}
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-56">
          <DropdownMenuLabel className="font-normal">
            <div className="flex flex-col gap-1">
              {username && (
                <p className="text-sm font-medium leading-none">{username}</p>
              )}
              <p className="text-xs text-muted-foreground leading-none">{email}</p>
            </div>
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuGroup>
            <DropdownMenuItem onSelect={() => navigate("/settings/profile")}>
              <User />
              <span>Profile</span>
            </DropdownMenuItem>
            <DropdownMenuItem onSelect={() => navigate("/settings/security")}>
              <Shield />
              <span>Security</span>
            </DropdownMenuItem>
            <DropdownMenuItem onSelect={() => navigate("/settings/billing")}>
              <CreditCard />
              <span>Billing</span>
            </DropdownMenuItem>
          </DropdownMenuGroup>
          <DropdownMenuSeparator />
          <DropdownMenuItem asChild>
            <a href="mailto:support@supersuit.tech">
              <LifeBuoy />
              <span>Support</span>
            </a>
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuCheckboxItem
            checked={theme === "dark"}
            onCheckedChange={toggleTheme}
          >
            <Moon />
            <span>Dark Mode</span>
          </DropdownMenuCheckboxItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={handleSignOut} variant="destructive">
            <LogOut />
            <span>Sign Out</span>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
  );
}
