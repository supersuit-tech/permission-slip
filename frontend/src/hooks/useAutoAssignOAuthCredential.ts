import { useEffect, useRef } from "react";
import { useSearchParams } from "react-router-dom";
import { toast } from "sonner";
import {
  useAgentConnectorCredential,
  useAssignAgentConnectorCredential,
} from "./useAgentConnectorCredential";

/**
 * After an OAuth flow completes, the backend redirects back with
 * `oauth_connection_id` as a query parameter. This hook reads that
 * parameter and auto-assigns the new OAuth connection to the current
 * agent+connector — unless a credential is already assigned.
 *
 * Mount this on the agent connector page (ConnectorCredentialsSection).
 */
export function useAutoAssignOAuthCredential(
  agentId: number,
  connectorId: string,
) {
  const [searchParams, setSearchParams] = useSearchParams();
  const { binding, isLoading: bindingLoading } =
    useAgentConnectorCredential(agentId, connectorId);
  const { assign } = useAssignAgentConnectorCredential();
  const firedRef = useRef(false);

  useEffect(() => {
    if (firedRef.current) return;
    if (bindingLoading) return;

    const connectionId = searchParams.get("oauth_connection_id");
    if (!connectionId) return;

    firedRef.current = true;

    // Clean up the param from the URL
    searchParams.delete("oauth_connection_id");
    setSearchParams(searchParams, { replace: true });

    // Skip if agent already has a credential assigned
    if (binding?.credential_id || binding?.oauth_connection_id) return;

    assign({ agentId, connectorId, oauthConnectionId: connectionId })
      .then(() => {
        toast.success("OAuth connection assigned to this agent.");
      })
      .catch((err) => {
        toast.error(
          err instanceof Error
            ? err.message
            : "Could not auto-assign credential — please select it manually.",
        );
      });
  }, [bindingLoading]); // eslint-disable-line react-hooks/exhaustive-deps -- run once when binding loads
}
