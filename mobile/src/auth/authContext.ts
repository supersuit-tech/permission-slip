/**
 * Shared AuthContext instance used by both the real AuthProvider and MockAuthProvider.
 * Extracted so useAuth() works regardless of which provider is mounted.
 */
import { createContext } from "react";
import type { AuthState } from "./types";

export const AuthContext = createContext<AuthState | undefined>(undefined);
