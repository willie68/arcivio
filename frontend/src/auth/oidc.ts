export const CLIENT_ID = "arcivio-web";
export const AUTH_BASE = "/auth";
const VERIFIER_KEY = "arcivio_pkce_verifier";
const STATE_KEY = "arcivio_oidc_state";
const NONCE_KEY = "arcivio_oidc_nonce";
const TOKEN_KEY = "arcivio_access_token";

function randomString(bytes: number): string {
  const buf = new Uint8Array(bytes);
  crypto.getRandomValues(buf);
  return base64Url(buf);
}

function base64Url(bytes: Uint8Array): string {
  let bin = "";
  bytes.forEach((b) => {
    bin += String.fromCharCode(b);
  });
  return btoa(bin).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

async function s256(verifier: string): Promise<string> {
  const data = new TextEncoder().encode(verifier);
  const hash = await crypto.subtle.digest("SHA-256", data);
  return base64Url(new Uint8Array(hash));
}

export function redirectUri(): string {
  return `${window.location.origin}/callback`;
}

export async function startAuthorization(): Promise<void> {
  const verifier = randomString(48);
  const state = randomString(16);
  const nonce = randomString(16);
  sessionStorage.setItem(VERIFIER_KEY, verifier);
  sessionStorage.setItem(STATE_KEY, state);
  sessionStorage.setItem(NONCE_KEY, nonce);
  const challenge = await s256(verifier);
  const params = new URLSearchParams({
    client_id: CLIENT_ID,
    redirect_uri: redirectUri(),
    response_type: "code",
    scope: "openid profile",
    state,
    nonce,
    code_challenge: challenge,
    code_challenge_method: "S256",
  });
  window.location.assign(`${AUTH_BASE}/authorize?${params.toString()}`);
}

export async function login(username: string, password: string): Promise<{ status: string; redirectTo?: string }> {
  const res = await fetch(`${AUTH_BASE}/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ username, password }),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data.message || data.error_description || "Anmeldung fehlgeschlagen");
  }
  return data;
}

export async function changePassword(oldPassword: string, newPassword: string): Promise<{ status: string; redirectTo?: string }> {
  const res = await fetch(`${AUTH_BASE}/change-password`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ oldPassword, newPassword }),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data.message || data.error_description || "Passwortänderung fehlgeschlagen");
  }
  return data;
}

export async function exchangeCode(code: string, state: string): Promise<void> {
  const expected = sessionStorage.getItem(STATE_KEY);
  if (!expected || expected !== state) {
    throw new Error("Ungültiger OIDC-State");
  }
  const verifier = sessionStorage.getItem(VERIFIER_KEY);
  if (!verifier) {
    throw new Error("PKCE-Verifier fehlt – bitte erneut anmelden");
  }
  const body = new URLSearchParams({
    grant_type: "authorization_code",
    code,
    redirect_uri: redirectUri(),
    client_id: CLIENT_ID,
    code_verifier: verifier,
  });
  const res = await fetch(`${AUTH_BASE}/token`, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body,
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data.error_description || data.error || "Token-Austausch fehlgeschlagen");
  }
  sessionStorage.setItem(TOKEN_KEY, data.access_token);
  sessionStorage.removeItem(VERIFIER_KEY);
  sessionStorage.removeItem(STATE_KEY);
  sessionStorage.removeItem(NONCE_KEY);
}

export function getAccessToken(): string | null {
  return sessionStorage.getItem(TOKEN_KEY);
}

export function hasPendingAuthRequest(): boolean {
  return Boolean(sessionStorage.getItem(VERIFIER_KEY));
}

export function clearSession(): void {
  sessionStorage.removeItem(TOKEN_KEY);
  sessionStorage.removeItem(VERIFIER_KEY);
  sessionStorage.removeItem(STATE_KEY);
  sessionStorage.removeItem(NONCE_KEY);
}

export async function logout(): Promise<void> {
  try {
    await fetch(`${AUTH_BASE}/logout`, { method: "POST", credentials: "include" });
  } catch {
    // ignore
  }
  clearSession();
}

export async function changeOwnPassword(oldPassword: string, newPassword: string): Promise<void> {
  const token = getAccessToken();
  if (!token) {
    throw new Error("not authenticated");
  }
  const res = await fetch("/api/v1/me/password", {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ oldPassword, newPassword }),
  });
  if (res.status === 401) {
    clearSession();
    throw new Error("not authenticated");
  }
  if (!res.ok) {
    const data = await res.json().catch(() => ({}));
    throw new Error(data.message || data.key || "password-change-failed");
  }
}

export type Me = {
  id: string;
  username: string;
  firstName: string;
  lastName: string;
  email: string;
  roles: string[];
  mustChangePassword: boolean;
  lastLogin?: string | null;
};

export type SettingsUser = {
  id: string;
  username: string;
  firstName: string;
  lastName: string;
  email: string;
  roles: string[];
  mustChangePassword: boolean;
  lastLogin?: string | null;
};

export type SettingsUserPage = {
  items: SettingsUser[];
  total: number;
  page: number;
  pageSize: number;
};

export async function listUsers(
  page: number,
  pageSize: number,
  sort = "username",
  order: "asc" | "desc" = "asc",
  prefix = "",
): Promise<SettingsUserPage> {
  const token = getAccessToken();
  if (!token) {
    throw new Error("not authenticated");
  }
  const params = new URLSearchParams({
    page: String(page),
    pageSize: String(pageSize),
    sort,
    order,
  });
  if ([...prefix.trim()].length >= 3) {
    params.set("prefix", prefix.trim());
  }
  const res = await fetch(`/api/v1/users?${params}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (res.status === 401) {
    clearSession();
    throw new Error("not authenticated");
  }
  if (res.status === 403) {
    throw new Error("forbidden");
  }
  if (!res.ok) {
    throw new Error("load-failed");
  }
  return res.json();
}

export type NewSettingsUser = {
  username: string;
  firstName: string;
  lastName: string;
  email: string;
  roles: string[];
};

export async function createUser(input: NewSettingsUser): Promise<{ user: SettingsUser; password: string }> {
  return apiJson("/api/v1/users", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateUser(id: string, input: NewSettingsUser): Promise<SettingsUser> {
  return apiJson(`/api/v1/users/${encodeURIComponent(id)}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export async function resetUserPassword(id: string): Promise<{ user: SettingsUser; password: string }> {
  return apiJson(`/api/v1/users/${encodeURIComponent(id)}/password-reset`, { method: "POST" });
}

export async function deleteUser(id: string): Promise<void> {
  await apiJson(`/api/v1/users/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export async function updateProfile(input: { firstName: string; lastName: string; email: string }): Promise<Me> {
  return apiJson("/api/v1/me", {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

async function apiJson(path: string, init: RequestInit): Promise<any> {
  const token = getAccessToken();
  if (!token) {
    throw new Error("not authenticated");
  }
  const res = await fetch(path, {
    ...init,
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
  });
  if (res.status === 401) {
    clearSession();
    throw new Error("not authenticated");
  }
  if (res.status === 204) {
    return undefined;
  }
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data.key || "request-failed");
  }
  return data;
}

export async function fetchMe(): Promise<Me> {
  const token = getAccessToken();
  if (!token) {
    throw new Error("not authenticated");
  }
  const res = await fetch("/api/v1/me", {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (res.status === 401) {
    clearSession();
    throw new Error("not authenticated");
  }
  if (!res.ok) {
    throw new Error("Benutzerprofil nicht lesbar");
  }
  return res.json();
}
