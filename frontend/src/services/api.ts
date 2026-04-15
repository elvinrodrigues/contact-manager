import type {
  ApiResponse,
  Contact,
  ContactFormData,
  ContactsListResponse,
  CreateContactResponse,
} from "@/types/contact";

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`/api${url}`, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });

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
  list: (page = 1, limit = 10) =>
    request<ContactsListResponse>(`/contacts?page=${page}&limit=${limit}`).then((res) => {
      if (res && res.contacts) res.contacts = res.contacts.map(mapContact);
      return res;
    }),

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
    console.log("deleted contacts response", res);
    if (res && res.contacts) res.contacts = res.contacts.map(mapContact);
    return res;
  }),
};
