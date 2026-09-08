import {
  formatStandingApprovalConstraints,
  type ConstraintMode as SharedConstraintMode,
} from "@permission-slip/constraints-format";

export {
  comparisonOpLabel,
  DATA_WINDOW_NAMESPACE_KEY,
  formatDataWindowConstraint,
  isPatternWrapper,
  metaConstraintLabel,
  META_NAMESPACE_KEY,
} from "@permission-slip/constraints-format";

export type ConstraintMode = SharedConstraintMode;

export interface ParsedConstraint {
  name: string;
  mode: ConstraintMode;
  value: string;
  negated?: boolean;
  comparisonOp?: "lte" | "gte" | "lt" | "gt";
  scenarioIndex?: number;
  href?: string;
  verified?: boolean;
}

function lineToParsedConstraint(line: {
  label: string;
  mode: ConstraintMode;
  value: string;
  negated?: boolean;
  comparisonOp?: "lte" | "gte" | "lt" | "gt";
  href?: string;
  verified?: boolean;
}): ParsedConstraint {
  const scenarioMatch = /^Scenario (\d+): /.exec(line.label);
  return {
    name: line.label,
    mode: line.mode,
    value: line.mode === "wildcard" ? "*" : line.value,
    negated: line.negated,
    comparisonOp: line.comparisonOp,
    scenarioIndex: scenarioMatch
      ? Number.parseInt(scenarioMatch[1] ?? "0", 10) - 1
      : undefined,
    href: line.href,
    verified: line.verified,
  };
}

/** Flatten standing-approval constraints into display rows (params + $meta + $data_window). */
export function parseStandingApprovalConstraints(
  constraints: Record<string, unknown> | null | undefined,
  resourceDetails?: Record<string, unknown> | null,
): ParsedConstraint[] {
  return formatStandingApprovalConstraints(constraints, resourceDetails).map(
    lineToParsedConstraint,
  );
}
