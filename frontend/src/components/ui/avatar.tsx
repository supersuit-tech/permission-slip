import { forwardRef } from "react";
import { cn } from "@/lib/utils";

function getInitials(name?: string | null, email?: string | null): string {
  const trimmedName = name?.trim();
  if (trimmedName) {
    const parts = trimmedName.split(/\s+/);
    const first = parts[0];
    const last = parts[parts.length - 1];
    if (parts.length >= 2 && first && last) {
      return (first.charAt(0) + last.charAt(0)).toUpperCase();
    }
    if (first) return first.charAt(0).toUpperCase();
  }
  const trimmedEmail = email?.trim();
  if (trimmedEmail) {
    return trimmedEmail.charAt(0).toUpperCase();
  }
  return "?";
}

const avatarStyles =
  "flex size-8 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground text-sm font-medium";

const Avatar = forwardRef<
  HTMLButtonElement,
  { name?: string | null; email?: string | null; as?: "button" | "span" } & React.ComponentProps<"button">
>(({ name, email, as: Tag = "button", className, ...props }, ref) => {
  const initials = getInitials(name, email);

  if (Tag === "span") {
    return (
      <span data-slot="avatar" className={cn(avatarStyles, className)}>
        {initials}
      </span>
    );
  }

  return (
    <button
      ref={ref}
      data-slot="avatar"
      type="button"
      className={cn(
        avatarStyles,
        "cursor-pointer select-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring",
        className
      )}
      {...props}
    >
      {initials}
    </button>
  );
});

Avatar.displayName = "Avatar";

export { Avatar, getInitials };
