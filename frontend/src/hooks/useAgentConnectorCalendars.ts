import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { useAgentConnectorCredential } from "./useAgentConnectorCredential";

export type RemoteCalendarRow = Record<string, unknown>;

export function useAgentConnectorCalendars(agentId: number, connectorId: string) {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const { binding, isCredentialBindingPending } =
    useAgentConnectorCredential(agentId, connectorId);

  const hasCredential = !!(
    binding?.oauth_connection_id ?? binding?.credential_id
  );

  const query = useQuery({
    queryKey: ["agent-connector-calendars", agentId, connectorId],
    staleTime: 60_000,
    queryFn: async () => {
      if (!accessToken) throw new Error("Missing access token");
      const { data, error } = await client.GET(
        "/v1/agents/{agent_id}/connectors/{connector_id}/calendars",
        {
          headers: { Authorization: `Bearer ${accessToken}` },
          params: {
            path: { agent_id: agentId, connector_id: connectorId },
          },
        },
      );
      if (error) throw new Error("Failed to load calendars");
      const rows = data?.data;
      if (!Array.isArray(rows)) return [] as RemoteCalendarRow[];
      return rows as RemoteCalendarRow[];
    },
    enabled: !!accessToken && agentId > 0 && !!connectorId && hasCredential,
  });

  return {
    calendars: query.data ?? [],
    isLoading: query.isLoading,
    isFetching: query.isFetching,
    error: query.isError ? query.error : null,
    hasCredential,
    isCredentialBindingPending,
    refetch: query.refetch,
  };
}
