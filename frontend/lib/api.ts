export type ApiError = {
  code: string;
  message: string;
};

export type ApiEnvelope<T> = {
  success: boolean;
  data?: T;
  error?: ApiError;
  request_id?: string;
};

const TOKEN_KEY = "grokforge.access_token";
const REFRESH_KEY = "grokforge.refresh_token";

export function getAccessToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(TOKEN_KEY);
}

export function getRefreshToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(REFRESH_KEY);
}

export function setTokens(access: string, refresh: string) {
  localStorage.setItem(TOKEN_KEY, access);
  localStorage.setItem(REFRESH_KEY, refresh);
}

export function clearTokens() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(REFRESH_KEY);
}

async function parseJSON<T>(res: Response): Promise<ApiEnvelope<T>> {
  const body = (await res.json()) as ApiEnvelope<T>;
  return body;
}

export async function api<T>(
  path: string,
  init: RequestInit = {},
  auth = false,
): Promise<ApiEnvelope<T>> {
  const headers = new Headers(init.headers || {});
  if (!headers.has("Content-Type") && init.body) {
    headers.set("Content-Type", "application/json");
  }
  if (auth) {
    const token = getAccessToken();
    if (token) headers.set("Authorization", `Bearer ${token}`);
  }
  const res = await fetch(path, { ...init, headers });
  return parseJSON<T>(res);
}

export type SetupStatus = {
  completed: boolean;
  completed_at: string | null;
  checks: Record<string, string>;
};

export type LoginData = {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  token_type: string;
  user: { id: number; username: string };
};

export type MeData = {
  id: number;
  username: string;
  must_reset_password: boolean;
};

export type SystemInfo = {
  name: string;
  version: string;
  setup_completed: boolean;
  time: string;
};
