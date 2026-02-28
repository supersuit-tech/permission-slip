import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";

export function useVAPIDKey() {
  const { session } = useAuth();
  const accessToken = session?.access_token;

  const tokenRef = useRef(accessToken);
  if (accessToken) {
    tokenRef.current = accessToken;
  }

  const query = useQuery({
    queryKey: ["vapid-key"],
    queryFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.GET("/v1/config/vapid-key", {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (error) throw new Error("Failed to fetch VAPID key");
      return data;
    },
    enabled: !!accessToken,
    staleTime: Infinity, // VAPID key doesn't change
  });

  return {
    vapidKey: query.data?.public_key ?? null,
    isLoading: query.isLoading,
  };
}
