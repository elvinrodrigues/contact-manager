import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { contactsApi } from "@/services/api";
import type { ContactFormData } from "@/types/contact";
import { toast } from "sonner";

export function useContacts(page: number, limit = 10) {
  return useQuery({
    queryKey: ["contacts", page, limit],
    queryFn: () => contactsApi.list(page, limit),
  });
}

export function useSearchContacts(query: string) {
  return useQuery({
    queryKey: ["contacts", "search", query],
    queryFn: () => contactsApi.search(query),
    enabled: query.length >= 2,
  });
}

export function useDeletedContacts(page: number, limit = 10) {
  return useQuery({
    queryKey: ["contacts", "deleted", page, limit],
    queryFn: () => contactsApi.listDeleted(page, limit),
  });
}

export function useCreateContact() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: ContactFormData) => contactsApi.create(data),
    onSuccess: (res) => {
      if (res.status === "created") {
        toast.success("Contact created successfully");
        qc.invalidateQueries({ queryKey: ["contacts"] });
      }
    },
    onError: (err: Error) => toast.error(err.message),
  });
}

export function useUpdateContact() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<ContactFormData> }) =>
      contactsApi.update(id, data),
    onSuccess: () => {
      toast.success("Contact updated");
      qc.invalidateQueries({ queryKey: ["contacts"] });
    },
    onError: (err: Error) => toast.error(err.message),
  });
}

export function useDeleteContact() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => contactsApi.delete(id),
    onMutate: async (id) => {
      await qc.cancelQueries({ queryKey: ["contacts"] });
      toast.success("Contact deleted");
    },
    onSettled: () => {
      qc.invalidateQueries({ queryKey: ["contacts"] });
    },
    onError: (err: Error) => toast.error(err.message),
  });
}

export function useRestoreContact() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => contactsApi.restore(id),
    onMutate: async (id) => {
      await qc.cancelQueries({ queryKey: ["contacts"] });
      toast.success("Contact restored");
    },
    onSettled: () => {
      qc.invalidateQueries({ queryKey: ["contacts"] });
    },
    onError: (err: Error) => toast.error(err.message),
  });
}
