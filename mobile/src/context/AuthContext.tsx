import { useQueryClient } from "@tanstack/react-query";
import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";

import { api, setSessionExpiredHandler } from "@/lib/api";
import { tokenStore } from "@/lib/storage";

export interface User {
  id: string;
  name: string;
  email: string;
  currency: string;
  timezone: string;
}

interface AuthResponse {
  user: User;
  access_token: string;
  refresh_token: string;
}

interface AuthContextValue {
  user: User | null;
  loading: boolean; // true selama aplikasi mengecek sesi tersimpan
  login: (email: string, password: string) => Promise<void>;
  register: (name: string, email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  updateUser: (u: User) => void;
  deleteAccount: (password: string) => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  // Saat aplikasi dibuka: jika ada sesi tersimpan, ambil profil dari server.
  useEffect(() => {
    setSessionExpiredHandler(() => {
      queryClient.clear();
      setUser(null);
    });

    let cancelled = false;
    (async () => {
      try {
        if (await tokenStore.getRefresh()) {
          const res = await api.get<{ user: User }>("/auth/me");
          if (!cancelled) setUser(res.user);
        }
      } catch {
        // belum login, sesi habis, atau server tidak terjangkau: tampilkan halaman login
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();

    return () => {
      cancelled = true;
      setSessionExpiredHandler(null);
    };
  }, [queryClient]);

  const finishAuth = useCallback(
    async (res: AuthResponse) => {
      await tokenStore.set(res.access_token, res.refresh_token);
      queryClient.clear(); // buang cache milik akun sebelumnya
      setUser(res.user);
    },
    [queryClient]
  );

  const login = useCallback(
    async (email: string, password: string) => {
      const res = await api.post<AuthResponse>(
        "/auth/login",
        { email, password },
        { auth: false }
      );
      await finishAuth(res);
    },
    [finishAuth]
  );

  const register = useCallback(
    async (name: string, email: string, password: string) => {
      const res = await api.post<AuthResponse>(
        "/auth/register",
        { name, email, password },
        { auth: false }
      );
      await finishAuth(res);
    },
    [finishAuth]
  );

  const logout = useCallback(async () => {
    try {
      const refresh = await tokenStore.getRefresh();
      if (refresh)
        await api.post(
          "/auth/logout",
          { refresh_token: refresh },
          { auth: false }
        );
    } catch {
      // tetap lanjut keluar secara lokal walau server tidak terjangkau
    }
    await tokenStore.clear();
    queryClient.clear();
    setUser(null);
  }, [queryClient]);

  const deleteAccount = useCallback(
    async (password: string) => {
      await api.delete("/auth/me", { password });
      await tokenStore.clear();
      queryClient.clear();
      setUser(null);
    },
    [queryClient]
  );
  const value = useMemo(
    () => ({
      user,
      loading,
      login,
      register,
      logout,
      updateUser: setUser,
      deleteAccount,
    }),
    [user, loading, login, register, logout, deleteAccount]
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth harus dipakai di dalam AuthProvider");
  return ctx;
}
