const API_BASE_URL = "http://localhost:8080";

async function getErrorMessage(
  response: Response,
  fallbackMessage: string,
): Promise<string> {
  const contentType = response.headers.get("content-type") || "";

  if (contentType.includes("application/json")) {
    const error = await response.json().catch(() => ({}));
    if (error && typeof error.message === "string" && error.message.trim()) {
      return error.message;
    }
  }

  const text = await response.text().catch(() => "");
  if (text.trim()) {
    return text;
  }

  return fallbackMessage;
}

/* -------------------- TYPES -------------------- */

export interface Contact {
  id: number;
  name: string;
  phone: string;
  email?: string;
  category_id?: number;
  deleted_at?: string | null;
}

export interface PaginatedContacts {
  contacts: Contact[];
  page: number;
  limit: number;
  total: number;
}

export interface CreateContactPayload {
  name: string;
  phone: string;
  email?: string;
  category_id?: number;
}

export interface CreateContactResponse {
  status: "created" | "duplicate";
  contact?: Contact;
  duplicates?: Contact[];
}

/* -------------------- CREATE CONTACT -------------------- */

export async function createContact(
  data: CreateContactPayload,
): Promise<CreateContactResponse> {
  const response = await fetch(`${API_BASE_URL}/contacts`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(data),
  });

  if (!response.ok) {
    throw new Error(
      await getErrorMessage(
        response,
        `Failed to create contact (${response.status})`,
      ),
    );
  }

  return response.json();
}

/* -------------------- FORCE CREATE CONTACT -------------------- */

export async function createContactForce(
  data: CreateContactPayload,
): Promise<Contact> {
  const response = await fetch(`${API_BASE_URL}/contacts/force`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(data),
  });

  if (!response.ok) {
    throw new Error(
      await getErrorMessage(
        response,
        `Failed to create contact (${response.status})`,
      ),
    );
  }

  return response.json();
}

/* -------------------- GET CONTACTS -------------------- */

export async function getContacts(): Promise<Contact[]> {
  const response = await fetch(`${API_BASE_URL}/contacts`, {
    method: "GET",
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch contacts (${response.status})`);
  }

  const data: PaginatedContacts = await response.json();

  return data.contacts || [];
}

/* -------------------- SEARCH CONTACTS -------------------- */

export async function searchContacts(query: string): Promise<Contact[]> {
  const response = await fetch(
    `${API_BASE_URL}/contacts/search?q=${encodeURIComponent(query)}`,
    {
      method: "GET",
    },
  );

  if (!response.ok) {
    throw new Error(`Failed to search contacts (${response.status})`);
  }

  const data: PaginatedContacts = await response.json();

  return data.contacts || [];
}

/* -------------------- DELETE CONTACT -------------------- */

export async function deleteContact(id: number): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/contacts/${id}`, {
    method: "DELETE",
  });

  if (!response.ok) {
    throw new Error(
      await getErrorMessage(
        response,
        `Failed to delete contact (${response.status})`,
      ),
    );
  }
}

/* -------------------- GET DELETED CONTACTS -------------------- */

export async function getDeletedContacts(): Promise<Contact[]> {
  const response = await fetch(`${API_BASE_URL}/contacts/deleted`, {
    method: "GET",
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch deleted contacts (${response.status})`);
  }

  const data: PaginatedContacts = await response.json();

  return data.contacts || [];
}

/* -------------------- RESTORE CONTACT -------------------- */

export async function restoreContact(id: number): Promise<Contact> {
  const response = await fetch(`${API_BASE_URL}/contacts/${id}/restore`, {
    method: "PATCH",
  });

  if (!response.ok) {
    throw new Error(
      await getErrorMessage(
        response,
        `Failed to restore contact (${response.status})`,
      ),
    );
  }

  return response.json();
}

/* -------------------- GET DUPLICATE CONTACTS -------------------- */

export async function getDuplicateContacts(): Promise<Contact[]> {
  const response = await fetch(`${API_BASE_URL}/contacts/duplicates`, {
    method: "GET",
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch duplicate contacts (${response.status})`);
  }

  const data: PaginatedContacts = await response.json();

  return data.contacts || [];
}
