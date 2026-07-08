/**
 * Builds the approval `context` object sent to POST /approvals/request.
 * `session_key` is stored verbatim and used by the server push wake to target
 * the agent's active OpenClaw session (see agentwake.SessionKeyFromApprovalContext).
 */

export interface ApprovalContextInput {
  description?: string;
  riskLevel?: string;
  sessionKey?: string;
}

export type ApprovalContext = Record<string, unknown>;

/** Returns undefined when no context fields were provided. */
export function buildApprovalContext(
  opts: ApprovalContextInput,
): ApprovalContext | undefined {
  const sessionKey = opts.sessionKey?.trim();
  const hasDescription = opts.description !== undefined && opts.description !== "";
  const hasRiskLevel = opts.riskLevel !== undefined && opts.riskLevel !== "";

  if (!hasDescription && !hasRiskLevel && !sessionKey) {
    return undefined;
  }

  return {
    ...(hasDescription ? { description: opts.description } : {}),
    ...(hasRiskLevel ? { risk_level: opts.riskLevel } : {}),
    ...(sessionKey ? { session_key: sessionKey } : {}),
  };
}
