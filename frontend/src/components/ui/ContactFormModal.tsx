import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import type { Contact, ContactFormData } from "@/types/contact";
import { useEffect } from "react";
import { CATEGORIES, DEFAULT_CATEGORY_ID } from "@/constants/categories";

const schema = z.object({
  name: z.string().trim().min(1, "Name is required").max(100),
  phone: z.string().trim().min(1, "Phone is required").max(20),
  email: z
    .string()
    .trim()
    .email("Invalid email")
    .max(255)
    .or(z.literal(""))
    .optional(),
  categoryId: z.coerce.number().optional(),
});

interface ContactFormModalProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (data: ContactFormData) => void;
  isSubmitting?: boolean;
  contact?: Contact | null;
}

export function ContactFormModal({
  open,
  onClose,
  onSubmit,
  isSubmitting,
  contact,
}: ContactFormModalProps) {
  const form = useForm<ContactFormData>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: "",
      phone: "",
      email: "",
      categoryId: DEFAULT_CATEGORY_ID,
    },
  });

  useEffect(() => {
    if (contact) {
      form.reset({
        name: contact.name,
        phone: contact.phone,
        email: contact.email || "",
        categoryId: contact.categoryId,
      });
    } else {
      form.reset({
        name: "",
        phone: "",
        email: "",
        categoryId: DEFAULT_CATEGORY_ID,
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [contact, open]);

  const handleSubmit = (data: ContactFormData) => {
    if (!data.email) delete data.email;
    if (!data.categoryId) data.categoryId = DEFAULT_CATEGORY_ID;

    if (contact) {
      const changed: Partial<ContactFormData> = {};
      if (data.name !== contact.name) changed.name = data.name;
      if (data.phone !== contact.phone) changed.phone = data.phone;
      if (data.email !== (contact.email || undefined))
        changed.email = data.email;
      if (data.categoryId !== contact.categoryId)
        changed.categoryId = data.categoryId;
      onSubmit(changed as ContactFormData);
    } else {
      onSubmit(data);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{contact ? "Edit Contact" : "New Contact"}</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form
            onSubmit={form.handleSubmit(handleSubmit)}
            className="space-y-4"
          >
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Name</FormLabel>
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="phone"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Phone</FormLabel>
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="email"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Email (optional)</FormLabel>
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="categoryId"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Category</FormLabel>
                  <FormControl>
                    <RadioGroup
                      value={String(field.value ?? DEFAULT_CATEGORY_ID)}
                      onValueChange={(v) => field.onChange(Number(v))}
                      className="flex flex-wrap gap-3"
                    >
                      {CATEGORIES.map((cat) => (
                        <div key={cat.id} className="flex items-center gap-1.5">
                          <RadioGroupItem
                            value={String(cat.id)}
                            id={`cat-${cat.id}`}
                          />
                          <label
                            htmlFor={`cat-${cat.id}`}
                            className="text-sm cursor-pointer"
                          >
                            {cat.name}
                          </label>
                        </div>
                      ))}
                    </RadioGroup>
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <div className="flex justify-end gap-2 pt-2">
              <Button type="button" variant="outline" onClick={onClose}>
                Cancel
              </Button>
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting ? "Saving..." : contact ? "Update" : "Create"}
              </Button>
            </div>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
