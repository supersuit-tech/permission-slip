import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import { useAgentConnectorCredential } from "./useAgentConnectorCredential";

export type RemoteUserRow = Record<string, unknown>;

/** Resolve the API base (mirrors api/client.ts normalizeBase). */
function apiBase(): string {
  const envUrl = import.meta.env.VITE_API_BASE_URL;
  if (!envUrl) return "/api";
  return (envUrl as string).replace(/\/v1\/?$/, "").replace(/\/$/, "");
}

export function useAgentConnectorUsers(agentId: number, connectorId: string) {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const { binding, isCredentialBindingPending } =
    useAgentConnectorCredential(agentId, connectorId);

  const hasCredential = !!(
    binding?.oauth_connection_id ?? binding?.credential_id
  );

  const query = useQuery({
    queryKey: ["agent-connector-users", agentId, connectorId],
    staleTime: 60_000,
    queryFn: async () => {
      if (!accessToken) throw new Error("Missing access token");
      const base = apiBase();
      const res = await fetch(
        `${base}/v1/agents/${agentId}/connectors/${connectorId}/users`,
        { headers: { Authorization: `Bearer ${accessToken}` } },
      );
      if (!res.ok) throw new Error("Failed to load users");
      const json = (await res.json()) as { data?: unknown[] };
      const rows = json?.data;
      if (!Array.isArray(rows)) return [] as RemoteUserRow[];
      return rows as RemoteUserRow[];
    },
    enabled: !!accessToken && agentId > 0 && !!connectorId && hasCredential,
  });

  return {
    users: query.data ?? [],
    isLoading: query.isLoading,
    isFetching: query.isFetching,
    error: query.isError ? query.error : null,
    hasCredential,
    isCredentialBindingPending,
    refetch: query.refetch,
  };
}
