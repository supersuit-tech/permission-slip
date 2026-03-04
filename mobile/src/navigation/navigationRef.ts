/**
 * Navigation ref for imperative navigation from outside React components
 * (e.g. notification tap handlers, deep link resolution).
 *
 * Lives in a separate module from RootNavigator to avoid circular
 * dependencies — hooks can import the ref without importing the
 * navigator component.
 */
import { createNavigationContainerRef } from "@react-navigation/native";
import type { RootStackParamList } from "./RootNavigator";

export const navigationRef = createNavigationContainerRef<RootStackParamList>();
