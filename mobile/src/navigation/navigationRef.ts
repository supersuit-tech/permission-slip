/**
 * Navigation ref for imperative navigation from outside React components
 * (e.g. notification tap handlers).
 */
import { createNavigationContainerRef } from "@react-navigation/native";
import type { RootStackParamList } from "./RootNavigator";

export const navigationRef = createNavigationContainerRef<RootStackParamList>();

/**
 * Navigate to a screen, waiting for the navigator to be ready if needed.
 * Safe to call from notification handlers where the navigator may not
 * yet be mounted.
 */
export function navigateWhenReady(
  name: keyof RootStackParamList,
  params?: RootStackParamList[keyof RootStackParamList],
) {
  if (navigationRef.isReady()) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    navigationRef.navigate(name as any, params as any);
  } else {
    // Retry after a short delay to allow the navigator to mount
    const timer = setInterval(() => {
      if (navigationRef.isReady()) {
        clearInterval(timer);
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        navigationRef.navigate(name as any, params as any);
      }
    }, 100);
    // Give up after 5 seconds
    setTimeout(() => clearInterval(timer), 5000);
  }
}
