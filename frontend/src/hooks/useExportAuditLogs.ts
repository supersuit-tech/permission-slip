import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { operations } from "@/api/schema";

/** Query parameters for the audit log export endpoint (from OpenAPI spec). */
export type ExportAuditLogsParams =
  operations["exportAuditLogs"]["parameters"]["query"];

/**
 * Fetches a single page of audit log export data.
 * Designed for compliance export / SIEM integration use cases.
 * Events are returned in chronological order (oldest first).
 */
export async function fetchExportPage(
  token: string,
  params: ExportAuditLogsParams,
) {
  const { data, error } = await client.GET("/v1/audit-logs", {
    headers: { Authorization: `Bearer ${token}` },
    params: {
      query: {
        since: params.since,
        ...(params.until ? { until: params.until } : {}),
        ...(params.event_type ? { event_type: params.event_type } : {}),
        ...(params.limit != null ? { limit: params.limit } : {}),
        ...(params.after ? { after: params.after } : {}),
      },
    },
  });
  if (error) throw new Error("Failed to export audit logs");
  return data;
}

/**
 * Hook for fetching a single page of audit log export data.
 * Pass `since` (RFC3339 string) to enable the query.
 *
 * Example:
 * ```ts
 * const { events, hasMore, nextCursor } = useExportAuditLogs({
 *   since: "2026-01-01T00:00:00Z",
 *   event_type: "approval.approved,approval.denied",
 * });
 * ```
 */
export function useExportAuditLogs(params: ExportAuditLogsParams) {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const userId = session?.user?.id;

  const query = useQuery({
    queryKey: [
      "audit-logs-export",
      userId ?? "",
      params.since,
      params.until,
      params.event_type,
      params.limit,
      params.after,
    ],
    queryFn: async () => {
      if (!accessToken) throw new Error("Missing access token");
      return fetchExportPage(accessToken, params);
    },
    enabled: !!accessToken && !!params.since,
  });

  return {
    events: query.data?.data ?? [],
    hasMore: query.data?.has_more ?? false,
    nextCursor: query.data?.next_cursor,
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to export audit logs. Please try again later."
      : null,
    refetch: query.refetch,
  };
}
