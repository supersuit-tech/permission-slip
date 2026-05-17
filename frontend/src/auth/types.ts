/** Client-side auth error (API errors + synthetic client errors). */
export type AuthError = {
  message: string;
  name?: string;
  status?: number;
  code?: string;
};

export type AuthStatus = "loading" | "unauthenticated" | "authenticated";

export type AppUser = {
  id: string;
  email?: string;
};

/** Session shape used across hooks (Bearer access token + stable user id). */
export type AppSession = {
  access_token: string;
  /** Absolute access-token expiry from the API (ISO-8601). */
  expires_at: string;
  user: AppUser;
};

export type AuthResult = { error: AuthError | null };

export interface AuthState {
  session: AppSession | null;
  user: AppUser | null;
  authStatus: AuthStatus;
  signInWithPassword: (email: string, password: string) => Promise<AuthResult>;
  signUpWithPassword: (email: string, password: string) => Promise<AuthResult>;
  updateEmail: (_newEmail: string) => Promise<AuthResult>;
  signOut: () => Promise<AuthResult>;
}
