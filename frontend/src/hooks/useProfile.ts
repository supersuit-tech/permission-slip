import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type Profile = components["schemas"]["ProfileResponse"];

export function useProfile() {
  const { session } = useAuth();
  const accessToken = session?.access_token;

  // Keep the latest access token in a ref so the query key stays stable
  // across AAL promotions (e.g. MFA enrollment upgrades AAL1 → AAL2 and
  // issues a new access token). Without this, React Query treats every
  // token change as a brand-new query — returning isLoading:true, which
  // causes App.tsx to show <LoadingFallback/> and unmount the entire
  // component tree (including any open MFA dialogs).
  const tokenRef = useRef(accessToken);
  if (accessToken) {
    tokenRef.current = accessToken;
  }

  const userId = session?.user?.id;

  const { data, isLoading } = useQuery({
    // Key on the user id (stable across token refreshes / AAL promotions).
    queryKey: ["profile", userId ?? ""],
    queryFn: async () => {
      // Always read the latest token from the ref at fetch time.
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error, response } = await client.GET("/v1/profile", {
        headers: { Authorization: `Bearer ${token}` },
      });
      // 404 means the user has no profile yet — needs onboarding.
      if (response.status === 404) {
        return { profile: null as Profile | null, needsOnboarding: true };
      }
      if (error) throw new Error("Failed to load profile");
      return { profile: data as Profile | null, needsOnboarding: false };
    },
    enabled: !!accessToken,
    // Profile data rarely changes — inherit the global 5min staleTime.
  });

  return {
    profile: data?.profile ?? null,
    needsOnboarding: data?.needsOnboarding ?? false,
    isLoading,
  };
}
