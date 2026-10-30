import { API_URL } from "@/config";
import { tokenStore } from "@/lib/storage";

export class ApiError extends Error {
  status: number;
  code: string;
  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

// Dipanggil saat refresh token ditolak server, supaya UI kembali ke halaman login.
let sessionExpiredHandler: (() => void) | null = null;
export function setSessionExpiredHandler(fn: (() => void) | null) {
  sessionExpiredHandler = fn;
}

const TIMEOUT_MS = 15000;

async function rawFetch(path: string, init: RequestInit): Promise<Response> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), TIMEOUT_MS);
  try {
    return await fetch(`${API_URL}${path}`, {
      ...init,
      signal: controller.signal,
    });
  } catch {
    throw new ApiError(
      0,
      "network_error",
      "Tidak dapat terhubung ke server. Periksa koneksi Anda."
    );
  } finally {
    clearTimeout(timer);
  }
}

// Beberapa request yang kena 401 bersamaan hanya boleh memicu SATU refresh.
// Ini penting karena refresh token dirotasi: pemakaian kedua pasti ditolak server.
let refreshing: Promise<boolean> | null = null;

function refreshTokens(): Promise<boolean> {
  if (!refreshing) {
    refreshing = (async () => {
      const refresh = await tokenStore.getRefresh();
      if (!refresh) return false;
      const res = await rawFetch("/auth/refresh", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refresh_token: refresh }),
      });
      if (res.status === 401) return false; // sesi memang sudah tidak valid
      if (!res.ok)
        throw new ApiError(
          res.status,
          "refresh_failed",
          "Gagal memperbarui sesi"
        );
      const data = await res.json();
      await tokenStore.set(data.access_token, data.refresh_token);
      return true;
    })().finally(() => {
      refreshing = null;
    });
  }
  return refreshing;
}

interface Options {
  method?: string;
  body?: unknown;
  auth?: boolean;
  headers?: Record<string, string>;
}

async function request<T>(
  path: string,
  opts: Options = {},
  retried = false
): Promise<T> {
  const { method = "GET", body, auth = true, headers = {} } = opts;

  const h: Record<string, string> = { Accept: "application/json", ...headers };
  if (body !== undefined) h["Content-Type"] = "application/json";
  if (auth) {
    const token = await tokenStore.getAccess();
    if (token) h.Authorization = `Bearer ${token}`;
  }

  const res = await rawFetch(path, {
    method,
    headers: h,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  let data: any = null;
  if (res.status !== 204) {
    try {
      data = await res.json();
    } catch {
      // body kosong atau bukan JSON
    }
  }

  // Refresh hanya jika server bilang token-nya bermasalah (bukan 401 karena password salah, dll.)
  if (
    res.status === 401 &&
    auth &&
    !retried &&
    data?.error?.code === "unauthorized"
  ) {
    const ok = await refreshTokens(); // error jaringan dilempar ke atas tanpa logout
    if (ok) return request<T>(path, opts, true);
    await tokenStore.clear();
    sessionExpiredHandler?.();
    throw new ApiError(
      401,
      "session_expired",
      "Sesi berakhir, silakan login kembali"
    );
  }

  if (res.status === 204) return undefined as T;
  if (!res.ok) {
    throw new ApiError(
      res.status,
      data?.error?.code ?? "unknown_error",
      data?.error?.message ?? "Terjadi kesalahan pada server"
    );
  }
  return data as T;
}

export const api = {
  get: <T>(path: string, opts?: Omit<Options, "method" | "body">) =>
    request<T>(path, { ...opts, method: "GET" }),
  post: <T>(
    path: string,
    body?: unknown,
    opts?: Omit<Options, "method" | "body">
  ) => request<T>(path, { ...opts, method: "POST", body }),
  patch: <T>(
    path: string,
    body?: unknown,
    opts?: Omit<Options, "method" | "body">
  ) => request<T>(path, { ...opts, method: "PATCH", body }),
  delete: <T>(
    path: string,
    body?: unknown,
    opts?: Omit<Options, "method" | "body">
  ) => request<T>(path, { ...opts, method: "DELETE", body }),
};
