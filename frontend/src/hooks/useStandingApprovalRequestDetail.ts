import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type StandingApprovalRequestDetail =
  components["schemas"]["StandingApprovalRequest"];

export function useStandingApprovalRequestDetail(requestId: string | null) {
  const { session } = useAuth();

  const query = useQuery({
    queryKey: ["standing-approval-requests", requestId],
    queryFn: async (): Promise<StandingApprovalRequestDetail> => {
      if (!session?.access_token || !requestId) {
        throw new Error("Missing request");
      }
      const { data, error } = await client.GET(
        "/v1/standing-approval-requests/{request_id}",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          params: { path: { request_id: requestId } },
        },
      );
      if (error) throw new Error("Failed to load rule proposal");
      return data;
    },
    enabled: !!session?.access_token && !!requestId,
  });

  return {
    request: query.data,
    isLoading: query.isLoading,
    error: query.isError,
  };
}
