export interface StandingApprovalFallthrough {
  reason: string;
  message: string;
}

export function parseStandingApprovalFallthrough(
  details: unknown,
): StandingApprovalFallthrough | null {
  if (!details || typeof details !== "object" || Array.isArray(details)) {
    return null;
  }
  const raw = (details as Record<string, unknown>).standing_approval_fallthrough;
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    return null;
  }
  const message = (raw as Record<string, unknown>).message;
  const reason = (raw as Record<string, unknown>).reason;
  if (typeof message !== "string" || message.length === 0) {
    return null;
  }
  return {
    reason: typeof reason === "string" ? reason : "",
    message,
  };
}
