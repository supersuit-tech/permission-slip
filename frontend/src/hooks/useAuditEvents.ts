import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { AuditEvent } from "@/lib/auditEvents";

export type { AuditEvent };

export interface AuditEventFilters {
  agent_id?: number;
  event_type?: string;
  outcome?: AuditEvent["outcome"];
}

/**
 * Builds the query key array used by both useAuditEvents and useInfiniteAuditEvents.
 * Shared so cache invalidation patterns stay consistent.
 */
export function auditEventsQueryKey(
  prefix: string,
  userId: string,
  filters: AuditEventFilters,
  limit: number,
) {
  return [prefix, userId, filters.agent_id, filters.event_type, filters.outcome, limit];
}

/**
 * Fetches a single page of audit events from the API.
 * Shared between useAuditEvents (single page) and useInfiniteAuditEvents (cursor-based).
 */
export async function fetchAuditEventsPage(
  token: string,
  filters: AuditEventFilters,
  limit: number,
  after?: string,
) {
  const { data, error } = await client.GET("/v1/audit-events", {
    headers: { Authorization: `Bearer ${token}` },
    params: {
      query: {
        limit,
        ...(after ? { after } : {}),
        ...(filters.agent_id != null && { agent_id: filters.agent_id }),
        ...(filters.event_type && { event_type: filters.event_type }),
        ...(filters.outcome && { outcome: filters.outcome }),
      },
    },
  });
  if (error) throw new Error("Failed to load activity");
  return data;
}

export function useAuditEvents(filters: AuditEventFilters = {}, limit = 20) {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const userId = session?.user?.id;

  // Keep the latest token in a ref so the query key stays stable across
  // token refreshes (e.g. Supabase re-issues tokens on AAL promotion).
  const tokenRef = useRef(accessToken);
  if (accessToken) {
    tokenRef.current = accessToken;
  }

  const query = useQuery({
    queryKey: auditEventsQueryKey("audit-events", userId ?? "", filters, limit),
    queryFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      return fetchAuditEventsPage(token, filters, limit);
    },
    enabled: !!accessToken,
    refetchInterval: 30_000,
  });

  return {
    events: query.data?.data ?? [],
    hasMore: query.data?.has_more ?? false,
    nextCursor: query.data?.next_cursor,
    retention: query.data?.retention ?? null,
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load activity. Please try again later."
      : null,
    refetch: query.refetch,
  };
}
