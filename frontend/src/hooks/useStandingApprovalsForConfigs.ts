import { useMemo } from "react";
import { useQueries } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { StandingApproval } from "@/hooks/useStandingApprovals";

const pageLimit = 100;

async function fetchAllStandingApprovalsForConfig(
  accessToken: string,
  configId: string,
): Promise<StandingApproval[]> {
  const all: StandingApproval[] = [];
  let after: string | undefined;
  for (;;) {
    const { data, error } = await client.GET("/v1/standing-approvals", {
      headers: { Authorization: `Bearer ${accessToken}` },
      params: {
        query: {
          status: "all",
          source_action_configuration_id: configId,
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
}

/**
 * Loads standing approvals (all statuses) per action configuration using the
 * server-side source_action_configuration_id filter. Used on the connector
 * config page to show standing-approval state per row.
 */
export function useStandingApprovalsForConfigs(configIds: string[]) {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const userId = session?.user?.id;

  const sortedIds = useMemo(
    () => [...new Set(configIds.filter((id) => id.length > 0))].sort(),
    [configIds],
  );

  const results = useQueries({
    queries: sortedIds.map((configId) => ({
      queryKey: ["standing-approvals-for-config", userId ?? "", configId],
      queryFn: async () => {
        const token = accessToken;
        if (!token) throw new Error("Missing access token");
        return fetchAllStandingApprovalsForConfig(token, configId);
      },
      enabled: !!accessToken && sortedIds.length > 0,
    })),
    combine: (queries) => {
      const map = new Map<string, StandingApproval[]>();
      let isLoading = false;
      let error: string | null = null;
      for (let i = 0; i < sortedIds.length; i++) {
        const q = queries[i];
        if (!q) continue;
        if (q.isLoading) isLoading = true;
        if (q.isError && !error) {
          error =
            "Unable to load standing approvals. Please try again later.";
        }
        map.set(sortedIds[i] ?? "", (q.data as StandingApproval[] | undefined) ?? []);
      }
      return {
        byConfigId: map,
        isLoading,
        error,
        refetch: () =>
          Promise.all(queries.map((q) => q.refetch())).then(() => undefined),
      };
    },
  });

  return results;
}
