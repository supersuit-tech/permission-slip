import { useState } from "react";
import { Asterisk, Lock, Regex, ShieldCheck } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  comparisonOpLabel,
  type ConstraintMode,
  type ParsedConstraint,
  parseStandingApprovalConstraints,
} from "@/lib/constraints";
import { UnrestrictedBadge } from "@/pages/agents/connectors/UnrestrictedBadge";

/** Maximum number of constraints shown before collapsing with "+N more". */
const VISIBLE_LIMIT = 2;

/** Maximum characters for a constraint value before truncating. */
const VALUE_TRUNCATE_LENGTH = 24;

const modeIcon: Record<ConstraintMode, React.ReactNode> = {
  fixed: <Lock className="size-3" />,
  pattern: <Regex className="size-3" />,
  wildcard: <Asterisk className="size-3" />,
};

function truncate(value: string, max: number): string {
  if (value.length <= max) return value;
  return value.slice(0, max - 1) + "\u2026";
}

function ConstraintBadge({ constraint }: { constraint: ParsedConstraint }) {
  const isVerified = constraint.name.startsWith("Verified ");
  const displayValue = constraint.comparisonOp
    ? `${comparisonOpLabel(constraint.comparisonOp)} ${truncate(constraint.value, VALUE_TRUNCATE_LENGTH)}`
    : constraint.mode === "wildcard"
      ? "any"
      : constraint.negated
        ? `not ${truncate(constraint.value, VALUE_TRUNCATE_LENGTH)}`
        : truncate(constraint.value, VALUE_TRUNCATE_LENGTH);

  return (
    <Badge
      variant="outline"
      className="gap-1 font-mono text-xs"
      title={`${constraint.name}: ${constraint.value} (${constraint.mode})`}
    >
      {isVerified ? <ShieldCheck className="size-3" /> : modeIcon[constraint.mode]}
      <span className="font-sans font-medium">{constraint.name}</span>
      <span className="text-muted-foreground">{displayValue}</span>
    </Badge>
  );
}

interface ConstraintsSummaryProps {
  constraints: Record<string, unknown> | null | undefined;
  unrestricted?: boolean;
}

export function ConstraintsSummary({
  constraints,
  unrestricted,
}: ConstraintsSummaryProps) {
  const [expanded, setExpanded] = useState(false);
  if (unrestricted) {
    return <UnrestrictedBadge />;
  }
  const parsed = parseStandingApprovalConstraints(constraints);

  if (parsed.length === 0) {
    return (
      <span className="text-muted-foreground text-xs italic">
        No constraints
      </span>
    );
  }

  const visible = expanded ? parsed : parsed.slice(0, VISIBLE_LIMIT);
  const remaining = parsed.length - VISIBLE_LIMIT;

  return (
    <div className="flex flex-wrap items-center gap-1">
      {visible.map((c) => (
        <ConstraintBadge key={c.name} constraint={c} />
      ))}
      {!expanded && remaining > 0 && (
        <button
          type="button"
          className="text-muted-foreground hover:text-foreground text-xs underline-offset-2 hover:underline"
          onClick={() => setExpanded(true)}
        >
          +{remaining} more
        </button>
      )}
      {expanded && parsed.length > VISIBLE_LIMIT && (
        <button
          type="button"
          className="text-muted-foreground hover:text-foreground text-xs underline-offset-2 hover:underline"
          onClick={() => setExpanded(false)}
        >
          show less
        </button>
      )}
    </div>
  );
}
