import { useState } from "react";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";

export function useDeleteAccount() {
  const { session } = useAuth();
  const [isDeleting, setIsDeleting] = useState(false);

  async function deleteAccount() {
    const accessToken = session?.access_token;
    if (!accessToken) throw new Error("Missing access token");

    setIsDeleting(true);
    try {
      const { error } = await client.DELETE("/v1/profile", {
        headers: { Authorization: `Bearer ${accessToken}` },
        body: { confirmation: "DELETE" },
      });
      if (error) throw new Error("Failed to delete account");
    } finally {
      setIsDeleting(false);
    }
  }

  return { deleteAccount, isDeleting };
}
