import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../auth/AuthContext";
import client from "../api/client";
import { getApiErrorMessage } from "../api/errors";
import type { components } from "../api/schema";

export type ApprovalBulkGroupSummary =
  components["schemas"]["ApprovalBulkGroupSummary"];

export function useApprovalBulkGroup(bulkGroupId: string | undefined) {
  const { session } = useAuth();
  const tokenRef = useRef(session?.access_token);
  tokenRef.current = session?.access_token;

  return useQuery({
    queryKey: ["approval-bulk-group", bulkGroupId ?? ""],
    queryFn: async (): Promise<ApprovalBulkGroupSummary> => {
      const token = tokenRef.current;
      if (!token || !bulkGroupId) throw new Error("Missing token or group id");
      const { data, error } = await client.GET("/v1/approval-groups/{group_id}", {
        headers: { Authorization: `Bearer ${token}` },
        params: { path: { group_id: bulkGroupId } },
      });
      if (error) {
        throw new Error(getApiErrorMessage(error, "Failed to load bulk group"));
      }
      return data;
    },
    enabled: !!session?.access_token && !!bulkGroupId,
  });
}
