import type {
  ApiResponse,
  Contact,
  ContactFormData,
  ContactsListResponse,
  CreateContactResponse,
} from "@/types/contact";
import { getToken, removeToken } from "@/lib/token";

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };

  const token = getToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`/api${url}`, {
    headers,
    ...options,
  });

  // Global 401 handler — clear token and redirect to login
  if (res.status === 401) {
    removeToken();
    window.location.href = "/login";
    throw new Error("Session expired");
  }

  let json: any;

  try {
    json = await res.json();
  } catch {
    throw new Error("Invalid JSON");
  }

  if (!res.ok) {
    throw new Error(json?.error || "Request failed");
  }

  if (json.error) {
    throw new Error(json.error);
  }

  return json.data;
}

// ── Auth API (public — no token needed) ──────────────────────────────────────

export const authApi = {
  signup: async (data: { name: string; email: string; password: string }) => {
    const res = await fetch("/api/auth/signup", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(data),
    });
    const json = await res.json();
    if (!res.ok) throw new Error(json?.error || "Signup failed");
    return json;
  },

  login: async (data: { email: string; password: string }) => {
    const res = await fetch("/api/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(data),
    });
    const json = await res.json();
    if (!res.ok) throw new Error(json?.error || "Login failed");
    return json;
  },

  verifyEmail: async (token: string) => {
    const res = await fetch(`/api/auth/verify?token=${encodeURIComponent(token)}`, {
      method: "GET",
    });
    const json = await res.json();
    if (!res.ok) throw new Error(json?.error || "Verification failed");
    return json;
  },

  forgotPassword: async (email: string) => {
    const res = await fetch("/api/auth/forgot-password", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email }),
    });
    const json = await res.json();
    if (!res.ok) throw new Error(json?.error || "Failed to process request");
    return json;
  },

  resetPassword: async (token: string, new_password: string) => {
    const res = await fetch("/api/auth/reset-password", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ token, new_password }),
    });
    const json = await res.json();
    if (!res.ok) throw new Error(json?.error || "Password reset failed");
    return json;
  },

  me: () => request<{ id: number; name: string; email: string; role: string }>("/auth/me"),
};

function mapContact(c: any): Contact {
  if (!c) return c;
  const mapped = { ...c };
  if ('category_id' in mapped) {
    mapped.categoryId = mapped.category_id;
    delete mapped.category_id;
  }
  return mapped;
}

export const contactsApi = {
  list: (page = 1, limit = 10, category = "all") => {
    const params = new URLSearchParams({ page: page.toString(), limit: limit.toString() });
    if (category && category !== "all") {
      params.append("category", category);
    }
    return request<ContactsListResponse>(`/contacts?${params.toString()}`).then((res) => {
      if (res && res.contacts) res.contacts = res.contacts.map(mapContact);
      return res;
    });
  },

  search: (q: string) =>
    request<ContactsListResponse>(`/contacts/search?q=${encodeURIComponent(q)}`).then((res) => {
      if (res && res.contacts) res.contacts = res.contacts.map(mapContact);
      return res;
    }),

  getById: (id: number) => request<Contact>(`/contacts/${id}`).then(mapContact),

  create: (data: ContactFormData) => {
    const payload = { ...data, category_id: data.categoryId };
    delete payload.categoryId;
    return request<CreateContactResponse>("/contacts", {
      method: "POST",
      body: JSON.stringify(payload),
    }).then((res) => {
      if (res && res.contact) res.contact = mapContact(res.contact);
      if (res && res.duplicates) res.duplicates = res.duplicates.map(mapContact);
      if (res && res.incoming) res.incoming = mapContact(res.incoming);
      return res;
    });
  },

  update: async (id: number, data: Partial<ContactFormData>): Promise<boolean> => {
    const payload: any = { ...data };
    if (payload.categoryId !== undefined) {
      payload.category_id = payload.categoryId;
      delete payload.categoryId;
    }
    
    await request(`/contacts/${id}`, {
      method: "PUT",
      body: JSON.stringify(payload),
    });
    return true;
  },

  delete: async (id: number): Promise<boolean> => {
    await request(`/contacts/${id}`, { method: "DELETE" });
    return true;
  },

  restore: async (id: number): Promise<boolean> => {
    await request(`/contacts/${id}/restore`, { method: "PATCH" });
    return true;
  },

  listDeleted: (page = 1, limit = 10) => request<ContactsListResponse>(`/contacts/deleted?page=${page}&limit=${limit}`).then((res) => {
    if (res && res.contacts) res.contacts = res.contacts.map(mapContact);
    return res;
  }),

  deletePermanent: async (id: number): Promise<void> => {
    await request<void>(`/contacts/${id}/permanent`, {
      method: "DELETE",
    });
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

export const adminApi = {
  listUsers: () => request<AdminUser[]>("/admin/users"),

  verifyUser: (id: number) =>
    request<void>(`/admin/users/${id}/verify`, { method: "PATCH" }),

  deleteUser: (id: number) =>
    request<void>(`/admin/users/${id}`, { method: "DELETE" }),
};
