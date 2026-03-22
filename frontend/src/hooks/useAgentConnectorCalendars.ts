import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";

const CALENDAR_CONNECTORS = new Set(["google", "microsoft"]);

export interface UserCalendarOption {
  id: string;
  name: string;
  description?: string;
  is_primary: boolean;
}

/**
 * Fetches calendars for the agent's bound Google or Microsoft credential.
 * Used to populate calendar_id select fields in action config UI.
 */
export function useAgentConnectorCalendars(
  connectorId: string,
  agentId: number,
  enabled: boolean,
) {
  const { session } = useAuth();

  const query = useQuery({
    queryKey: ["agent-connector-calendars", agentId, connectorId],
    queryFn: async (): Promise<UserCalendarOption[]> => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }
      const { data, error, response } = await client.GET(
        "/v1/agents/{agent_id}/connectors/{connector_id}/calendars",
        {
          params: {
            path: { agent_id: agentId, connector_id: connectorId },
          },
          headers: { Authorization: `Bearer ${session.access_token}` },
        },
      );
      if (error) {
        throw new Error(
          getApiErrorMessage(
            error,
            `Failed to load calendars (${response.status})`,
          ),
        );
      }
      const rows = data?.data;
      if (!Array.isArray(rows)) return [];

      const out: UserCalendarOption[] = [];
      for (const row of rows) {
        if (
          row &&
          typeof row === "object" &&
          typeof (row as { id?: unknown }).id === "string" &&
          typeof (row as { name?: unknown }).name === "string"
        ) {
          const r = row as {
            id: string;
            name: string;
            description?: string | null;
            is_primary?: boolean;
          };
          out.push({
            id: r.id,
            name: r.name,
            description: r.description ?? undefined,
            is_primary: Boolean(r.is_primary),
          });
        }
      }

      out.sort((a, b) => {
        if (a.is_primary !== b.is_primary) return a.is_primary ? -1 : 1;
        return a.name.localeCompare(b.name);
      });
      return out;
    },
    enabled:
      enabled &&
      !!session?.access_token &&
      agentId > 0 &&
      CALENDAR_CONNECTORS.has(connectorId),
    staleTime: 60_000,
  });

  return {
    calendars: query.data,
    isLoading: query.isLoading,
    isFetching: query.isFetching,
    error: query.error instanceof Error ? query.error.message : null,
    refetch: query.refetch,
  };
}
