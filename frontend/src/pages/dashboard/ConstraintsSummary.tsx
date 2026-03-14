import { useState } from "react";
import { Asterisk, Lock, Regex } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { isPatternWrapper } from "@/lib/constraints";

/** Maximum number of constraints shown before collapsing with "+N more". */
const VISIBLE_LIMIT = 2;

/** Maximum characters for a constraint value before truncating. */
const VALUE_TRUNCATE_LENGTH = 24;

type ConstraintMode = "fixed" | "pattern" | "wildcard";

interface ParsedConstraint {
  name: string;
  mode: ConstraintMode;
  value: string;
}

function parseConstraints(
  constraints: Record<string, unknown> | null | undefined,
): ParsedConstraint[] {
  if (!constraints || typeof constraints !== "object") return [];
  return Object.entries(constraints).map(([name, raw]) => {
    if (raw === "*") {
      return { name, mode: "wildcard" as const, value: "*" };
    }
    if (isPatternWrapper(raw)) {
      return { name, mode: "pattern" as const, value: raw.$pattern };
    }
    return { name, mode: "fixed" as const, value: String(raw) };
  });
}

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
  const displayValue =
    constraint.mode === "wildcard"
      ? "any"
      : truncate(constraint.value, VALUE_TRUNCATE_LENGTH);

  return (
    <Badge
      variant="outline"
      className="gap-1 font-mono text-xs"
      title={`${constraint.name}: ${constraint.value} (${constraint.mode})`}
    >
      {modeIcon[constraint.mode]}
      <span className="font-sans font-medium">{constraint.name}</span>
      <span className="text-muted-foreground">{displayValue}</span>
    </Badge>
  );
}

interface ConstraintsSummaryProps {
  constraints: Record<string, unknown> | null | undefined;
}

export function ConstraintsSummary({ constraints }: ConstraintsSummaryProps) {
  const [expanded, setExpanded] = useState(false);
  const parsed = parseConstraints(constraints);

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
