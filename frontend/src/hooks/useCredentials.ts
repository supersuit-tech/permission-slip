import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type CredentialSummary = components["schemas"]["CredentialSummary"];

export function useCredentials(options?: { enabled?: boolean }) {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const userId = session?.user?.id;
  const externalEnabled = options?.enabled ?? true;

  // Keep the latest token in a ref so the query key stays stable across
  // token refreshes (e.g. Supabase re-issues tokens on AAL promotion).
  const tokenRef = useRef(accessToken);
  if (accessToken) {
    tokenRef.current = accessToken;
  }

  const query = useQuery({
    queryKey: ["credentials", userId ?? ""],
    queryFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.GET("/v1/credentials", {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (error) throw new Error("Failed to load credentials");
      return data;
    },
    enabled: !!accessToken && externalEnabled,
  });

  return {
    credentials: query.data?.credentials ?? [],
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load credentials. Please try again later."
      : null,
    refetch: query.refetch,
  };
}
