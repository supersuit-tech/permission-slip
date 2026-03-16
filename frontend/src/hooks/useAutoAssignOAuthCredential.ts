import { useEffect, useRef } from "react";
import { useSearchParams } from "react-router-dom";
import { useTryAutoAssign } from "./useTryAutoAssign";

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
  const { tryAssign, bindingLoading } = useTryAutoAssign(agentId, connectorId);
  const firedRef = useRef(false);
  const searchParamsRef = useRef(searchParams);
  searchParamsRef.current = searchParams;

  useEffect(() => {
    if (firedRef.current) return;
    if (bindingLoading) return;

    const params = searchParamsRef.current;
    const connectionId = params.get("oauth_connection_id");
    if (!connectionId) return;

    firedRef.current = true;

    // Clean up the param from the URL (work on a copy to avoid mutating
    // the live URLSearchParams instance before React Router re-renders).
    const cleaned = new URLSearchParams(params);
    cleaned.delete("oauth_connection_id");
    setSearchParams(cleaned, { replace: true });

    // connectionId comes from the URL (set by our backend's OAuth redirect).
    // The assign endpoint validates ownership and provider match server-side.
    tryAssign({ oauthConnectionId: connectionId });
  }, [bindingLoading]); // eslint-disable-line react-hooks/exhaustive-deps -- run once when binding loads
}
