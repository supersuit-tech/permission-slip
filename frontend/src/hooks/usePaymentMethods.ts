import { useRef } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type PaymentMethod = components["schemas"]["PaymentMethod"];

export function usePaymentMethods() {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const userId = session?.user?.id;

  const tokenRef = useRef(accessToken);
  if (accessToken) {
    tokenRef.current = accessToken;
  }

  const query = useQuery({
    queryKey: ["payment-methods", userId ?? ""],
    queryFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.GET("/v1/payment-methods", {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (error) throw new Error("Failed to load payment methods");
      return data;
    },
    enabled: !!accessToken,
  });

  return {
    paymentMethods: query.data?.payment_methods ?? [],
    maxAllowed: query.data?.max_allowed ?? 0,
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load payment methods. Please try again later."
      : null,
    refetch: query.refetch,
  };
}

export function useCreateSetupIntent() {
  const { session } = useAuth();
  const tokenRef = useRef(session?.access_token);
  if (session?.access_token) {
    tokenRef.current = session.access_token;
  }

  const mutation = useMutation({
    mutationFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.POST(
        "/v1/payment-methods/setup-intent",
        {
          headers: { Authorization: `Bearer ${token}` },
        },
      );
      if (error) throw new Error("Failed to create setup intent");
      return data;
    },
  });

  return {
    createSetupIntent: mutation.mutateAsync,
    isLoading: mutation.isPending,
  };
}

export function useConfirmPaymentMethod() {
  const { session } = useAuth();
  const queryClient = useQueryClient();
  const userId = session?.user?.id;
  const tokenRef = useRef(session?.access_token);
  if (session?.access_token) {
    tokenRef.current = session.access_token;
  }

  const mutation = useMutation({
    mutationFn: async (params: {
      payment_method_id: string;
      label?: string;
      is_default: boolean;
    }) => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.POST("/v1/payment-methods", {
        headers: { Authorization: `Bearer ${token}` },
        body: params,
      });
      if (error) throw new Error("Failed to save payment method");
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["payment-methods", userId ?? ""],
      });
    },
  });

  return {
    confirmPaymentMethod: mutation.mutateAsync,
    isLoading: mutation.isPending,
  };
}

export function useUpdatePaymentMethod() {
  const { session } = useAuth();
  const queryClient = useQueryClient();
  const userId = session?.user?.id;
  const tokenRef = useRef(session?.access_token);
  if (session?.access_token) {
    tokenRef.current = session.access_token;
  }

  const mutation = useMutation({
    mutationFn: async ({
      id,
      ...body
    }: {
      id: string;
      label?: string;
      is_default?: boolean;
      per_transaction_limit?: number | null;
      monthly_limit?: number | null;
      clear_per_transaction_limit?: boolean;
      clear_monthly_limit?: boolean;
    }) => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.PATCH(
        "/v1/payment-methods/{id}",
        {
          headers: { Authorization: `Bearer ${token}` },
          params: { path: { id } },
          body,
        },
      );
      if (error) throw new Error("Failed to update payment method");
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["payment-methods", userId ?? ""],
      });
    },
  });

  return {
    updatePaymentMethod: mutation.mutateAsync,
    isLoading: mutation.isPending,
  };
}

export function useDeletePaymentMethod() {
  const { session } = useAuth();
  const queryClient = useQueryClient();
  const userId = session?.user?.id;
  const tokenRef = useRef(session?.access_token);
  if (session?.access_token) {
    tokenRef.current = session.access_token;
  }

  const mutation = useMutation({
    mutationFn: async (id: string) => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.DELETE(
        "/v1/payment-methods/{id}",
        {
          headers: { Authorization: `Bearer ${token}` },
          params: { path: { id } },
        },
      );
      if (error) throw new Error("Failed to delete payment method");
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["payment-methods", userId ?? ""],
      });
    },
  });

  return {
    deletePaymentMethod: mutation.mutateAsync,
    isLoading: mutation.isPending,
  };
}
