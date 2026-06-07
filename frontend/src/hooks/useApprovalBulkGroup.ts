import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type ApprovalBulkGroupSummary =
  components["schemas"]["ApprovalBulkGroupSummary"];

export function useApprovalBulkGroup(bulkGroupId: string | undefined) {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const tokenRef = useRef(accessToken);
  if (accessToken) {
    tokenRef.current = accessToken;
  }

  return useQuery({
    queryKey: ["approval-bulk-group", bulkGroupId ?? ""],
    queryFn: async (): Promise<ApprovalBulkGroupSummary> => {
      const token = tokenRef.current;
      if (!token || !bulkGroupId) {
        throw new Error("Missing token or group id");
      }
      const { data, error } = await client.GET("/v1/approval-groups/{group_id}", {
        headers: { Authorization: `Bearer ${token}` },
        params: { path: { group_id: bulkGroupId } },
      });
      if (error) throw new Error("Failed to load bulk approval group");
      return data;
    },
    enabled: !!accessToken && !!bulkGroupId,
  });
}
