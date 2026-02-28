/**
 * Known fields within agent metadata. The server stores metadata as an
 * open-ended JSONB object, so we keep the index signature for forward
 * compatibility while giving named fields proper types.
 */
export interface AgentMetadata {
  name?: string;
  [key: string]: unknown;
}

/**
 * Narrows an opaque metadata value (from the API) into AgentMetadata,
 * returning null if the value is not an object.
 */
export function parseAgentMetadata(
  raw: unknown,
): AgentMetadata | null {
  if (raw != null && typeof raw === "object" && !Array.isArray(raw)) {
    // Safe: narrowed to non-null, non-array object — matches AgentMetadata's index signature
    return raw as AgentMetadata;
  }
  return null;
}

/**
 * Returns a human-readable display name for an agent. Uses
 * `metadata.name` when available, otherwise falls back to "Agent <id>".
 */
export function getAgentDisplayName(agent: {
  agent_id: number;
  metadata?: unknown;
}): string {
  const meta = parseAgentMetadata(agent.metadata);
  if (meta?.name && meta.name.length > 0) {
    return meta.name;
  }
  return `Agent ${agent.agent_id}`;
}
