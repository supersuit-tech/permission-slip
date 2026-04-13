import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { StandingApproval } from "@/hooks/useStandingApprovals";

const pageLimit = 100;

/**
 * Loads standing approvals (all statuses) for the current user and groups them
 * by source_action_configuration_id. Used on the connector config page to show
 * standing-approval state per action configuration.
 */
export function useStandingApprovalsForConfigs(configIds: string[]) {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const userId = session?.user?.id;

  const sortedIds = useMemo(
    () => [...new Set(configIds.filter((id) => id.length > 0))].sort(),
    [configIds],
  );

  const query = useQuery({
    queryKey: ["standing-approvals-by-config-ids", userId ?? "", sortedIds.join(",")],
    queryFn: async () => {
      if (!accessToken) throw new Error("Missing access token");
      const all: StandingApproval[] = [];
      let after: string | undefined;
      for (;;) {
        const { data, error } = await client.GET("/v1/standing-approvals", {
          headers: { Authorization: `Bearer ${accessToken}` },
          params: {
            query: {
              status: "all",
              limit: pageLimit,
              ...(after ? { after } : {}),
            },
          },
        });
        if (error) throw new Error("Failed to load standing approvals");
        const batch = data?.data ?? [];
        all.push(...batch);
        if (!data?.has_more || !data.next_cursor) break;
        after = data.next_cursor;
        if (batch.length === 0) break;
      }
      return all;
    },
    enabled: !!accessToken && sortedIds.length > 0,
  });

  const byConfigId = useMemo(() => {
    const map = new Map<string, StandingApproval[]>();
    const idSet = new Set(sortedIds);
    const rows = query.data ?? [];
    for (const sa of rows) {
      const sid = sa.source_action_configuration_id;
      if (!sid || !idSet.has(sid)) continue;
      const list = map.get(sid);
      if (list) list.push(sa);
      else map.set(sid, [sa]);
    }
    return map;
  }, [query.data, sortedIds]);

  return {
    byConfigId,
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load standing approvals. Please try again later."
      : null,
    refetch: query.refetch,
  };
}
