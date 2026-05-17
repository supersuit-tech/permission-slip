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

export type AppSession = {
  access_token: string;
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
  signOut: () => Promise<AuthResult>;
}
