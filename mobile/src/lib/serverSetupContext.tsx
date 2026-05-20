import { createContext, useContext } from "react";

/**
 * Lets any unauthenticated screen (login, connection-error retry) open the
 * server URL setup overlay from App.tsx. Without this, a user stuck pointing
 * at a dead server would have no way to fix it short of reinstalling — and
 * on iOS, expo-secure-store uses Keychain, which persists across uninstalls,
 * so even reinstalling does not always reset the saved URL.
 */
type ServerSetupContextValue = {
  openServerSetup: () => void;
};

export const ServerSetupContext = createContext<ServerSetupContextValue>({
  openServerSetup: () => {},
});

export function useServerSetup(): ServerSetupContextValue {
  return useContext(ServerSetupContext);
}
