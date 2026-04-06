import type { User, Stats, LoginResponse } from "./types";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("token");
}

async function request<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const token = getToken();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...((options.headers as Record<string, string>) || {}),
  };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${API_URL}${path}`, {
    ...options,
    headers,
  });

  if (res.status === 401) {
    localStorage.removeItem("token");
    window.location.href = "/login";
    throw new Error("Unauthorized");
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || `Request failed: ${res.status}`);
  }

  return res.json();
}

export async function login(
  username: string,
  password: string
): Promise<string> {
  const data = await request<LoginResponse>("/api/login", {
    method: "POST",
    body: JSON.stringify({ username, password }),
  });
  localStorage.setItem("token", data.token);
  return data.token;
}

export async function getStats(): Promise<Stats> {
  return request<Stats>("/api/stats");
}

export async function getUsers(): Promise<User[]> {
  const data = await request<{ users: User[] }>("/api/users");
  return data.users;
}

export async function createUser(
  username: string,
  password: string
): Promise<void> {
  await request("/api/users", {
    method: "POST",
    body: JSON.stringify({ username, password }),
  });
}

export async function updateUser(
  username: string,
  data: { password?: string; enabled?: boolean }
): Promise<void> {
  await request(`/api/users/${encodeURIComponent(username)}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export async function deleteUser(username: string): Promise<void> {
  await request(`/api/users/${encodeURIComponent(username)}`, {
    method: "DELETE",
  });
}

export function logout() {
  localStorage.removeItem("token");
  window.location.href = "/login";
}

export function isAuthenticated(): boolean {
  return !!getToken();
}
