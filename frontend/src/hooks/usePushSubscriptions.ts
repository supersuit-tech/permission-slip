import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";

export function usePushSubscriptions() {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const userId = session?.user?.id;

  const tokenRef = useRef(accessToken);
  if (accessToken) {
    tokenRef.current = accessToken;
  }

  const query = useQuery({
    queryKey: ["push-subscriptions", userId ?? ""],
    queryFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.GET("/v1/push-subscriptions", {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (error) throw new Error("Failed to fetch push subscriptions");
      return data;
    },
    enabled: !!accessToken,
  });

  return {
    subscriptions: query.data?.subscriptions ?? [],
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load push subscriptions. Please try again later."
      : null,
  };
}
