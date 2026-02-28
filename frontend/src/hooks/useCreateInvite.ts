import { useMutation } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";
import { trackEvent, PostHogEvents } from "@/lib/posthog";

export type InviteResponse =
  components["schemas"]["CreateRegistrationInviteResponse"];

export function useCreateInvite() {
  const { session } = useAuth();

  const mutation = useMutation({
    mutationFn: async () => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { data, error } = await client.POST(
        "/v1/registration-invites",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          body: {},
        },
      );
      if (error) throw new Error("Failed to generate invite code");
      return data;
    },
    onSuccess: () => {
      trackEvent(PostHogEvents.INVITE_CREATED);
    },
  });

  return {
    createInvite: () => mutation.mutateAsync(),
    isLoading: mutation.isPending,
  };
}
