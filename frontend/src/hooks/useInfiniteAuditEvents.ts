import { useRef } from "react";
import { useInfiniteQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import type { AuditEvent } from "@/lib/auditEvents";
import {
  type AuditEventFilters,
  auditEventsQueryKey,
  fetchAuditEventsPage,
} from "./useAuditEvents";

export type { AuditEvent, AuditEventFilters };

export function useInfiniteAuditEvents(
  filters: AuditEventFilters = {},
  limit = 50,
) {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const userId = session?.user?.id;

  const tokenRef = useRef(accessToken);
  if (accessToken) {
    tokenRef.current = accessToken;
  }

  const query = useInfiniteQuery({
    queryKey: auditEventsQueryKey(
      "audit-events-infinite",
      userId ?? "",
      filters,
      limit,
    ),
    queryFn: async ({ pageParam }) => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      return fetchAuditEventsPage(token, filters, limit, pageParam);
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) =>
      lastPage?.has_more ? lastPage.next_cursor : undefined,
    enabled: !!accessToken,
    // No refetchInterval — useInfiniteQuery refetches ALL loaded pages,
    // which grows costly as the user loads more. The single-page
    // useAuditEvents (dashboard) has its own 30s polling.
  });

  const events: AuditEvent[] =
    query.data?.pages.flatMap((page) => page?.data ?? []) ?? [];

  return {
    events,
    hasNextPage: query.hasNextPage,
    isFetchingNextPage: query.isFetchingNextPage,
    fetchNextPage: query.fetchNextPage,
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load activity. Please try again later."
      : null,
    refetch: query.refetch,
  };
}
