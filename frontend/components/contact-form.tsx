"use client";

import { useState, useRef } from "react";
import { useForm, Controller } from "react-hook-form";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";

import { FieldGroup, Field, FieldLabel } from "@/components/ui/field";

import { Input } from "@/components/ui/input";

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

import { Spinner } from "@/components/ui/spinner";
import { toast } from "sonner";

import {
  Contact,
  createContact,
  CreateContactPayload,
} from "@/lib/api-service";
import { DuplicateDialog } from "@/components/duplicate-dialog";

type ContactFormValues = {
  name: string;
  phone: string;
  email?: string;
  category_id?: number;
};

const categories = [
  { id: 1, name: "Default" },
  { id: 2, name: "Family" },
  { id: 3, name: "Friends" },
  { id: 4, name: "Work" },
];

interface ContactFormProps {
  onContactCreated?: () => void;
}

export function ContactForm({ onContactCreated }: ContactFormProps) {
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);

  const [duplicateDialogOpen, setDuplicateDialogOpen] = useState(false);
  const [duplicates, setDuplicates] = useState<Contact[]>([]);

  const pendingContact = useRef<CreateContactPayload | null>(null);

  const {
    register,
    handleSubmit,
    reset,
    control,
    formState: { errors },
  } = useForm<ContactFormValues>({
    defaultValues: {
      name: "",
      phone: "",
      email: "",
      category_id: undefined,
    },
  });

  const onSubmit = async (data: ContactFormValues) => {
    setLoading(true);

    try {
      const result = await createContact(data);

      if (result.status === "created") {
        toast.success("Contact created successfully");

        reset();
        setOpen(false);

        onContactCreated?.();
      }

      if (result.status === "duplicate") {
        pendingContact.current = data;

        setDuplicates(result.duplicates || []);

        setDuplicateDialogOpen(true);
      }
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : "Failed to create contact",
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogTrigger asChild>
          <Button>Add Contact</Button>
        </DialogTrigger>

        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>Create New Contact</DialogTitle>
            <DialogDescription>
              Add a new contact to your contact list.
            </DialogDescription>
          </DialogHeader>

          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            {/* Name */}
            <FieldGroup>
              <Field>
                <FieldLabel>Name</FieldLabel>

                <Input
                  placeholder="John Doe"
                  {...register("name", { required: "Name is required" })}
                />

                {errors.name && (
                  <p className="text-sm text-red-500">{errors.name.message}</p>
                )}
              </Field>
            </FieldGroup>

            {/* Phone */}
            <FieldGroup>
              <Field>
                <FieldLabel>Phone</FieldLabel>

                <Input
                  placeholder="+91 0123456789"
                  {...register("phone", { required: "Phone is required" })}
                />

                {errors.phone && (
                  <p className="text-sm text-red-500">{errors.phone.message}</p>
                )}
              </Field>
            </FieldGroup>

            {/* Email */}
            <FieldGroup>
              <Field>
                <FieldLabel>Email</FieldLabel>

                <Input
                  type="email"
                  placeholder="john@example.com"
                  {...register("email")}
                />
              </Field>
            </FieldGroup>

            {/* Category */}
            <FieldGroup>
              <Field>
                <FieldLabel>Category</FieldLabel>

                <Controller
                  control={control}
                  name="category_id"
                  render={({ field }) => (
                    <Select
                      onValueChange={(value) => field.onChange(Number(value))}
                      value={field.value ? field.value.toString() : undefined}
                    >
                      <SelectTrigger>
                        <SelectValue placeholder="Select category" />
                      </SelectTrigger>

                      <SelectContent>
                        {categories.map((cat) => (
                          <SelectItem key={cat.id} value={cat.id.toString()}>
                            {cat.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  )}
                />
              </Field>
            </FieldGroup>

            {/* Buttons */}
            <div className="flex gap-2 justify-end pt-4">
              <Button
                type="button"
                variant="outline"
                onClick={() => {
                  setOpen(false);
                  reset();
                }}
                disabled={loading}
              >
                Cancel
              </Button>

              <Button type="submit" disabled={loading}>
                {loading ? (
                  <>
                    <Spinner className="mr-2 h-4 w-4" />
                    Creating...
                  </>
                ) : (
                  "Create Contact"
                )}
              </Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>

      <DuplicateDialog
        open={duplicateDialogOpen}
        duplicates={duplicates}
        onCancel={() => setDuplicateDialogOpen(false)}
      />
    </>
  );
}
