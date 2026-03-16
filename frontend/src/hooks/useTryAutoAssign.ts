import { useCallback, useRef } from "react";
import { toast } from "sonner";
import {
  useAgentConnectorCredential,
  useAssignAgentConnectorCredential,
} from "./useAgentConnectorCredential";

/**
 * Returns a function that attempts to auto-assign a credential (static or
 * OAuth) to the given agent+connector — but only if no credential is already
 * bound. Shows a toast on success or failure.
 *
 * Pass `agentId: 0` or `undefined` to disable (the returned function becomes
 * a no-op).
 */
export function useTryAutoAssign(
  agentId: number | undefined,
  connectorId: string,
) {
  const effectiveId = agentId ?? 0;
  const { binding, isLoading: bindingLoading } =
    useAgentConnectorCredential(effectiveId, connectorId);
  const { assign } = useAssignAgentConnectorCredential();
  const bindingRef = useRef(binding);
  bindingRef.current = binding;

  const tryAssign = useCallback(
    (params: { credentialId?: string; oauthConnectionId?: string }) => {
      if (!effectiveId || effectiveId <= 0) return;
      if (bindingRef.current?.credential_id || bindingRef.current?.oauth_connection_id) return;

      assign({
        agentId: effectiveId,
        connectorId,
        credentialId: params.credentialId,
        oauthConnectionId: params.oauthConnectionId,
      })
        .then(() => {
          toast.success(
            params.oauthConnectionId
              ? "OAuth connection assigned to this agent."
              : "Credential assigned to this agent.",
          );
        })
        .catch((err) => {
          toast.error(
            err instanceof Error
              ? err.message
              : "Could not auto-assign credential — please select it manually.",
          );
        });
    },
    [effectiveId, connectorId, assign],
  );

  return { tryAssign, bindingLoading };
}
