import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";

/** Active standing approvals linked to an action configuration (for UI copy). */
export function useLinkedStandingApprovalsForConfig(
  configId: string | undefined,
  enabled: boolean,
) {
  const { session } = useAuth();
  const accessToken = session?.access_token;

  return useQuery({
    queryKey: ["standing-approvals-linked", configId],
    queryFn: async () => {
      if (!accessToken || !configId) throw new Error("Missing context");
      const { data, error } = await client.GET("/v1/standing-approvals", {
        headers: { Authorization: `Bearer ${accessToken}` },
        params: {
          query: {
            status: "active",
            source_action_configuration_id: configId,
            limit: 100,
          },
        },
      });
      if (error) throw new Error("Failed to load linked standing approvals");
      return data;
    },
    enabled: !!accessToken && !!configId && enabled,
  });
}
