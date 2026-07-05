interface StandingApprovalInstanceScopeLineProps {
  label: string;
  className?: string;
}

export function StandingApprovalInstanceScopeLine({
  label,
  className,
}: StandingApprovalInstanceScopeLineProps) {
  return (
    <p
      className={
        className ??
        "text-muted-foreground rounded-md border bg-muted/40 px-3 py-2 text-sm"
      }
    >
      {label}
    </p>
  );
}
