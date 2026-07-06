function normalizeEmailAddress(raw: string): string {
  const trimmed = raw.trim();
  const match = /<([^>]+)>/.exec(trimmed);
  return (match?.[1] ?? trimmed).trim();
}

function primarySenderAddress(from: unknown): string | null {
  if (typeof from === "string" && from.length > 0) return normalizeEmailAddress(from);
  if (Array.isArray(from)) {
    const first = from.find((value): value is string => typeof value === "string" && value.length > 0);
    return first ? normalizeEmailAddress(first) : null;
  }
  return null;
}

/** Convert approval resource_details into constraint-matching metadata when possible. */
export function resourceDetailsToConstraintMeta(
  resourceDetails?: Record<string, unknown> | null,
): Record<string, unknown> | null {
  if (!resourceDetails) return null;

  const source =
    resourceDetails.in_reply_to && typeof resourceDetails.in_reply_to === "object"
      ? (resourceDetails.in_reply_to as Record<string, unknown>)
      : resourceDetails;

  const from = primarySenderAddress(source.from);
  if (!from) return null;

  return {
    sender: from,
    senders: [from],
    messages: [
      {
        from,
        to: source.to ?? [],
        cc: source.cc ?? [],
        bcc: source.bcc ?? [],
      },
    ],
  };
}
