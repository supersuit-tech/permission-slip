interface ApprovalSectionProps {
  label: string;
  children: React.ReactNode;
  className?: string;
}

/** Labeled section with card container — mirrors mobile approval detail layout. */
export function ApprovalSection({
  label,
  children,
  className,
}: ApprovalSectionProps) {
  return (
    <section className={className}>
      <h3 className="text-muted-foreground mb-2 text-xs font-semibold uppercase tracking-wide">
        {label}
      </h3>
      <div className="rounded-xl border bg-card p-3 shadow-sm sm:p-4">{children}</div>
    </section>
  );
}
