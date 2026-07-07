import {
  formatStandingApprovalConstraints,
  comparisonOpLabel,
  type ConstraintMode as SharedConstraintMode,
} from "@permission-slip/constraints-format";

export {
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
}

function lineToParsedConstraint(line: {
  label: string;
  mode: ConstraintMode;
  value: string;
  negated?: boolean;
  comparisonOp?: "lte" | "gte" | "lt" | "gt";
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
  };
}

/** Flatten standing-approval constraints into display rows (params + $meta + $data_window). */
export function parseStandingApprovalConstraints(
  constraints: Record<string, unknown> | null | undefined,
): ParsedConstraint[] {
  return formatStandingApprovalConstraints(constraints).map(lineToParsedConstraint);
}
