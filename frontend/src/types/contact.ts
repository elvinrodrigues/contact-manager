export interface Contact {
  id: number;
  name: string;
  phone: string;
  /** Absent when no address is recorded — the API omits the field. */
  email?: string;
  categoryId?: number;
  createdAt?: string;
  updatedAt?: string;
  deletedAt?: string;
  daysRemaining?: number;
}

export interface ContactFormData {
  name: string;
  phone: string;
  email?: string;
  categoryId?: number;
}

export interface ApiError {
  code: string;
  message: string;
}

export interface ApiResponse<T = unknown> {
  data: T | null;
  error: ApiError | null;
  message: string;
}

export interface ContactsListResponse {
  contacts: Contact[];
  page: number;
  limit: number;
  total: number;
}

export interface CreateContactResponse {
  status: "created" | "duplicate" | "deleted_duplicate";
  contact?: Contact;
  duplicates?: Contact[];
  incoming?: Contact;
}
