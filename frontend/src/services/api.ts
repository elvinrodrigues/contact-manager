import type {
  ApiResponse,
  Contact,
  ContactFormData,
  ContactsListResponse,
  CreateContactResponse,
} from "@/types/contact";
import { getToken, removeToken } from "@/lib/token";

/**
 * Where the API lives.
 *
 * In development this stays "/api" and Vite's proxy forwards to the local
 * backend (see vite.config.ts). In a deployed build there is no proxy, so
 * VITE_API_BASE_URL must point at the backend's public origin — otherwise every
 * request resolves against the static host and 404s.
 */
const API_BASE = (import.meta.env.VITE_API_BASE_URL ?? "/api").replace(/\/$/, "");

function apiUrl(path: string): string {
  return `${API_BASE}${path}`;
}

/** Everything the API returns, success or failure, uses this envelope. */
interface Envelope<T> {
  data: T | null;
  error: { code: string; message: string } | null;
  message: string;
}

/** An error carrying the API's machine-readable code and HTTP status. */
export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly code: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

/** Pulls a readable message out of an envelope, whatever shape it arrived in. */
function errorMessage(json: any, fallback: string): string {
  if (typeof json?.error === "string") return json.error; // older responses
  return json?.error?.message ?? json?.message ?? fallback;
}

function errorCode(json: any): string {
  return typeof json?.error === "object" && json?.error ? json.error.code : "unknown";
}

async function readJson(res: Response): Promise<any> {
  const text = await res.text();
  if (!text) return {};
  try {
    return JSON.parse(text);
  } catch {
    throw new ApiError("The server sent a response we could not read.", res.status, "invalid_json");
  }
}

interface RequestOptions extends RequestInit {
  /** Statuses to hand back as data instead of throwing (e.g. 409 duplicates). */
  expect?: number[];
}

async function request<T>(url: string, options?: RequestOptions): Promise<T> {
  const { expect = [], ...init } = options ?? {};

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...((init.headers as Record<string, string>) ?? {}),
  };

  const token = getToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(apiUrl(url), { ...init, headers });

  // A 401 means the session is gone — the token was revoked, expired, or the
  // account no longer exists. Clear it and start over.
  if (res.status === 401) {
    removeToken();
    window.location.href = "/login";
    throw new ApiError("Your session has ended. Please sign in again.", 401, "unauthorized");
  }

  const json = await readJson(res);

  if (!res.ok && !expect.includes(res.status)) {
    throw new ApiError(errorMessage(json, "Request failed"), res.status, errorCode(json));
  }

  return json.data as T;
}

/** Auth calls run without a token and surface their own errors. */
async function publicRequest<T>(url: string, body: unknown, fallback: string): Promise<T> {
  const res = await fetch(apiUrl(url), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });

  const json = await readJson(res);
  if (!res.ok) {
    throw new ApiError(errorMessage(json, fallback), res.status, errorCode(json));
  }
  return json as T;
}

// ── Auth API (public — no token needed) ──────────────────────────────────────

export const authApi = {
  signup: (data: { name: string; email: string; password: string }) =>
    publicRequest<Envelope<{ id: number; email: string }>>("/auth/signup", data, "Signup failed"),

  login: (data: { email: string; password: string }) =>
    publicRequest<Envelope<{ token: string }>>("/auth/login", data, "Login failed"),

  verifyEmail: async (token: string) => {
    const res = await fetch(apiUrl(`/auth/verify?token=${encodeURIComponent(token)}`));
    const json = await readJson(res);
    if (!res.ok) {
      throw new ApiError(errorMessage(json, "Verification failed"), res.status, errorCode(json));
    }
    return json;
  },

  forgotPassword: (email: string) =>
    publicRequest<Envelope<null>>("/auth/forgot-password", { email }, "Could not send the reset link"),

  resetPassword: (token: string, new_password: string) =>
    publicRequest<Envelope<null>>("/auth/reset-password", { token, new_password }, "Password reset failed"),

  me: () =>
    request<{
      id: number;
      name: string;
      email: string;
      role: string;
      is_verified: boolean;
    }>("/auth/me"),
};

/** The API speaks snake_case; the UI speaks camelCase. */
function mapContact(c: any): Contact {
  if (!c) return c;
  const { category_id, created_at, updated_at, deleted_at, ...rest } = c;
  return {
    ...rest,
    categoryId: category_id,
    createdAt: created_at,
    updatedAt: updated_at,
    deletedAt: deleted_at,
  };
}

/** An empty page is [] rather than null, but stay defensive about it. */
function mapList(res: ContactsListResponse): ContactsListResponse {
  return { ...res, contacts: (res?.contacts ?? []).map(mapContact) };
}

function toPayload(data: Partial<ContactFormData>): Record<string, unknown> {
  const { categoryId, ...rest } = data;
  const payload: Record<string, unknown> = { ...rest };
  if (categoryId !== undefined) payload.category_id = categoryId;
  return payload;
}

export const contactsApi = {
  list: (page = 1, limit = 10, category = "all") => {
    const params = new URLSearchParams({ page: String(page), limit: String(limit) });
    // The API expects a numeric category id; "all" means no filter.
    if (category && category !== "all") {
      params.append("category", category);
    }
    return request<ContactsListResponse>(`/contacts?${params}`).then(mapList);
  },

  search: (q: string, page = 1, limit = 20) => {
    const params = new URLSearchParams({ q, page: String(page), limit: String(limit) });
    return request<ContactsListResponse>(`/contacts/search?${params}`).then(mapList);
  },

  getById: (id: number) => request<Contact>(`/contacts/${id}`).then(mapContact),

  create: (data: ContactFormData) => {
    // A duplicate phone number is reported as 409 with the conflicting records
    // attached, so it is an expected outcome here rather than a failure — the
    // UI opens its merge dialog from this payload.
    return request<CreateContactResponse>("/contacts", {
      method: "POST",
      body: JSON.stringify(toPayload(data)),
      expect: [409],
    }).then((res) => {
      if (res?.contact) res.contact = mapContact(res.contact);
      if (res?.duplicates) res.duplicates = res.duplicates.map(mapContact);
      if (res?.incoming) res.incoming = mapContact(res.incoming);
      return res;
    });
  },

  // The API returns the stored contact, so the caller sees exactly what was
  // applied rather than assuming the request went through as sent.
  update: (id: number, data: Partial<ContactFormData>) =>
    request<Contact>(`/contacts/${id}`, {
      method: "PUT",
      body: JSON.stringify(toPayload(data)),
    }).then(mapContact),

  delete: async (id: number): Promise<void> => {
    await request<null>(`/contacts/${id}`, { method: "DELETE" });
  },

  restore: async (id: number): Promise<void> => {
    await request<null>(`/contacts/${id}/restore`, { method: "PATCH" });
  },

  listDeleted: (page = 1, limit = 10) => {
    const params = new URLSearchParams({ page: String(page), limit: String(limit) });
    return request<ContactsListResponse>(`/contacts/deleted?${params}`).then(mapList);
  },

  deletePermanent: async (id: number): Promise<void> => {
    await request<null>(`/contacts/${id}/permanent`, { method: "DELETE" });
  },

  getStats: () =>
    request<{
      total: number;
      deleted: number;
      added_this_week: number;
      recent: Contact[];
      categories: { name: string; count: number }[];
    }>("/contacts/stats").then((res) => {
      if (res && res.recent) res.recent = res.recent.map(mapContact);
      return res;
    }),
};

// ── Admin API ────────────────────────────────────────────────────────────────

export type AdminUser = {
  id: number;
  name: string;
  email: string;
  is_verified: boolean;
  role: string;
  created_at: string;
};

export type AdminUsersResponse = {
  users: AdminUser[];
  page: number;
  limit: number;
  total: number;
};

export const adminApi = {
  // The listing is paginated; it previously returned every account at once.
  listUsers: (page = 1, limit = 50) => {
    const params = new URLSearchParams({ page: String(page), limit: String(limit) });
    return request<AdminUsersResponse>(`/admin/users?${params}`).then((res) => ({
      ...res,
      users: res?.users ?? [],
    }));
  },

  verifyUser: (id: number) =>
    request<null>(`/admin/users/${id}/verify`, { method: "PATCH" }),

  deleteUser: (id: number) =>
    request<null>(`/admin/users/${id}`, { method: "DELETE" }),
};
