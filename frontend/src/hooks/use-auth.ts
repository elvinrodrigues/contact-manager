import { createContext, useContext, useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { getToken, setToken, removeToken } from "@/lib/token";
import { authApi } from "@/services/api";
import React from "react";

// ── Types ──────────────────────────────────────────────────────────────────────

export type UserProfile = {
  id: number;
  name: string;
  email: string;
  role: string;
};

type AuthContextType = {
  user: UserProfile | null;
  isAuthenticated: boolean;
  loading: boolean;
  login: (token: string) => Promise<void>;
  logout: () => void;
};

// ── Context ────────────────────────────────────────────────────────────────────

const AuthContext = createContext<AuthContextType>({
  user: null,
  isAuthenticated: false,
  loading: true,
  login: async () => {},
  logout: () => {},
});

// ── Provider ───────────────────────────────────────────────────────────────────

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<UserProfile | null>(null);
  const [loading, setLoading] = useState(true);

  const isAuthenticated = !!user;

  // On mount: if a token exists, fetch the user profile
  useEffect(() => {
    const token = getToken();
    if (!token) {
      setLoading(false);
      return;
    }

    authApi
      .me()
      .then((data) => setUser(data))
      .catch(() => {
        // Token is invalid/expired — clean up
        removeToken();
        setUser(null);
      })
      .finally(() => setLoading(false));
  }, []);

  // login: save token, fetch user profile, then let the caller navigate
  const login = useCallback(async (token: string) => {
    setToken(token);
    try {
      const data = await authApi.me();
      setUser(data);
    } catch {
      removeToken();
      setUser(null);
    }
  }, []);

  // logout: clear everything
  const logout = useCallback(() => {
    removeToken();
    setUser(null);
  }, []);

  return React.createElement(
    AuthContext.Provider,
    { value: { user, isAuthenticated, loading, login, logout } },
    children,
  );
}

// ── Hook ───────────────────────────────────────────────────────────────────────

export function useAuth() {
  return useContext(AuthContext);
}
