"use client";

import { useState, useRef, useEffect } from "react";
import { createPortal } from "react-dom";
import { useForm, Controller } from "react-hook-form";
import { Plus, User, Phone, Mail, Tag, ChevronDown, Check } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Spinner } from "@/components/ui/spinner";
import { toast } from "sonner";
import { cn } from "@/lib/utils";
import {
  Contact,
  createContact,
  CreateContactPayload,
} from "@/lib/api-service";
import { DuplicateDialog } from "@/components/duplicate-dialog";

// ─── Types ────────────────────────────────────────────────────────────────

type ContactFormValues = {
  name: string;
  phone: string;
  email?: string;
  category_id?: number;
};

// ─── Category options ─────────────────────────────────────────────────────

const CATEGORIES = [
  { id: 1, name: "Default", emoji: "📋" },
  { id: 2, name: "Family", emoji: "🏠" },
  { id: 3, name: "Friends", emoji: "🤝" },
  { id: 4, name: "Work", emoji: "💼" },
] as const;

// ─── Sub-components ───────────────────────────────────────────────────────

/** A labelled input row with an optional left icon and inline error */
function FormField({
  label,
  required,
  optional,
  icon: Icon,
  error,
  children,
}: {
  label: string;
  required?: boolean;
  optional?: boolean;
  icon: React.ElementType;
  error?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-1.5">
      {/* Label row */}
      <div className="flex items-baseline gap-1.5">
        <label className="text-sm font-medium text-foreground leading-none">
          {label}
        </label>
        {required && (
          <span className="text-destructive text-xs font-semibold leading-none">
            *
          </span>
        )}
        {optional && (
          <span className="text-muted-foreground/70 text-[11px] font-normal leading-none">
            optional
          </span>
        )}
      </div>

      {/* Input wrapper */}
      <div className="relative">
        {/* Left icon */}
        <div className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground/60">
          <Icon size={14} strokeWidth={2} />
        </div>

        {/* The actual input / select */}
        <div className="[&>*]:pl-9">{children}</div>
      </div>

      {/* Inline validation error */}
      {error && (
        <p className="flex items-center gap-1 text-[12px] font-medium text-destructive leading-snug">
          <span className="inline-block w-3.5 h-3.5 rounded-full bg-destructive/15 flex-shrink-0 flex items-center justify-center text-[9px] font-black">
            !
          </span>
          {error}
        </p>
      )}
    </div>
  );
}

/** Shared input base styles */
const inputBase = cn(
  "w-full h-9 rounded-xl border border-border bg-background",
  "text-sm text-foreground placeholder:text-muted-foreground/60",
  "shadow-sm",
  "transition-[border-color,box-shadow] duration-150",
  "focus:outline-none focus:border-ring focus:ring-2 focus:ring-ring/20",
  "disabled:opacity-50 disabled:cursor-not-allowed",
);

const inputError =
  "border-destructive focus:border-destructive focus:ring-destructive/20";

// ─── Props ────────────────────────────────────────────────────────────────

interface ContactFormProps {
  onContactCreated?: () => void;
}

// ─── Main component ───────────────────────────────────────────────────────

export function ContactForm({ onContactCreated }: ContactFormProps) {
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);

  const [duplicateDialogOpen, setDuplicateDialogOpen] = useState(false);
  const [duplicates, setDuplicates] = useState<Contact[]>([]);

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const pendingContact = useRef<CreateContactPayload | null>(null);

  const {
    register,
    handleSubmit,
    reset,
    control,
    formState: { errors, isSubmitted },
  } = useForm<ContactFormValues>({
    defaultValues: {
      name: "",
      phone: "",
      email: "",
      category_id: 1,
    },
  });

  // ── Submit ──────────────────────────────────────────────────────────────

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

  const handleClose = () => {
    if (loading) return;
    setOpen(false);
    reset();
  };

  // ── Render ──────────────────────────────────────────────────────────────

  return (
    <>
      {/* ── Trigger ──────────────────────────────────────────────────────── */}
      <Dialog
        open={open}
        onOpenChange={(v) => (v ? setOpen(true) : handleClose())}
      >
        <DialogTrigger asChild>
          <Button
            className={cn(
              "gap-2 h-9 px-4 rounded-xl text-sm font-semibold",
              "shadow-sm shadow-primary/20",
              "transition-all duration-150 active:scale-[0.97]",
            )}
          >
            <Plus size={15} strokeWidth={2.5} />
            New Contact
          </Button>
        </DialogTrigger>

        {/* ── Dialog ───────────────────────────────────────────────────── */}
        <DialogContent className="p-0 gap-0 sm:max-w-[460px]">
          {/* ── Dialog header ─────────────────────────────────────────── */}
          <div className="px-6 pt-6 pb-5 border-b border-border">
            {/* Accent dot */}
            <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center mb-3">
              <User size={18} className="text-primary" strokeWidth={2} />
            </div>

            <DialogHeader className="gap-0.5 text-left p-0">
              <DialogTitle className="text-base font-semibold text-foreground">
                Add New Contact
              </DialogTitle>
              <DialogDescription className="text-sm text-muted-foreground">
                Fill in the details below. Fields marked{" "}
                <span className="text-destructive font-semibold">*</span> are
                required.
              </DialogDescription>
            </DialogHeader>
          </div>

          {/* ── Form body ─────────────────────────────────────────────── */}
          <form
            onSubmit={handleSubmit(onSubmit)}
            noValidate
            className="px-6 py-5 space-y-4"
          >
            {/* Name */}
            <FormField
              label="Full Name"
              required
              icon={User}
              error={errors.name?.message}
            >
              <input
                type="text"
                placeholder="Jane Doe"
                autoComplete="name"
                autoFocus
                className={cn(
                  inputBase,
                  isSubmitted && errors.name && inputError,
                )}
                {...register("name", {
                  required: "Full name is required",
                  minLength: {
                    value: 2,
                    message: "Name must be at least 2 characters",
                  },
                })}
              />
            </FormField>

            {/* Phone */}
            <FormField
              label="Phone Number"
              required
              icon={Phone}
              error={errors.phone?.message}
            >
              <input
                type="tel"
                placeholder="+1 (555) 000-0000"
                autoComplete="tel"
                className={cn(
                  inputBase,
                  "font-mono tracking-tight",
                  isSubmitted && errors.phone && inputError,
                )}
                {...register("phone", {
                  required: "Phone number is required",
                  minLength: {
                    value: 6,
                    message: "Enter a valid phone number",
                  },
                })}
              />
            </FormField>

            {/* Email */}
            <FormField
              label="Email Address"
              optional
              icon={Mail}
              error={errors.email?.message}
            >
              <input
                type="email"
                placeholder="jane@example.com"
                autoComplete="email"
                className={cn(
                  inputBase,
                  isSubmitted && errors.email && inputError,
                )}
                {...register("email", {
                  pattern: {
                    value: /^[^\s@]+@[^\s@]+\.[^\s@]+$/,
                    message: "Enter a valid email address",
                  },
                })}
              />
            </FormField>

            {/* Category */}
            <FormField label="Category" optional icon={Tag}>
              <Controller
                control={control}
                name="category_id"
                render={({ field }) => (
                  <CategorySelect
                    value={field.value}
                    onChange={field.onChange}
                  />
                )}
              />
            </FormField>

            {/* ── Action buttons ──────────────────────────────────────── */}
            <div className="flex items-center justify-end gap-3 pt-2">
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={handleClose}
                disabled={loading}
                className="h-9 px-4 rounded-xl text-sm"
              >
                Cancel
              </Button>

              <Button
                type="submit"
                disabled={loading}
                className={cn(
                  "h-9 px-5 rounded-xl text-sm font-semibold gap-2",
                  "shadow-sm shadow-primary/20",
                  "transition-all duration-150 active:scale-[0.97]",
                  "min-w-[130px]",
                )}
              >
                {loading ? (
                  <>
                    <Spinner className="h-3.5 w-3.5" />
                    Creating…
                  </>
                ) : (
                  <>
                    <Plus size={14} strokeWidth={2.5} />
                    Create Contact
                  </>
                )}
              </Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>

      {/* ── Duplicate dialog ─────────────────────────────────────────────── */}
      <DuplicateDialog
        open={duplicateDialogOpen}
        duplicates={duplicates}
        onCancel={() => setDuplicateDialogOpen(false)}
      />
    </>
  );
}

// ─── CategorySelect ───────────────────────────────────────────────────────
// A lightweight custom select so we get the left-icon padding for free
// without fighting with the Radix SelectTrigger's own layout.

function CategorySelect({
  value,
  onChange,
}: {
  value?: number;
  onChange: (v: number) => void;
}) {
  const [open, setOpen] = useState(false);
  const [menuStyle, setMenuStyle] = useState<React.CSSProperties>({});
  const triggerRef = useRef<HTMLButtonElement>(null);
  // Guard against SSR — createPortal needs document.body
  const [mounted, setMounted] = useState(false);
  // Effective value — fall back to 1 (Default) when nothing is selected
  const effectiveValue = value ?? 1;
  const selected = CATEGORIES.find((c) => c.id === effectiveValue)!;

  useEffect(() => {
    setMounted(true);
  }, []);

  // Close on Escape
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open]);

  const openMenu = () => {
    if (!triggerRef.current) return;

    const rect = triggerRef.current.getBoundingClientRect();
    const GAP = 6;
    const MENU_MAX_H = 224; // matches max-h-56 below
    const spaceBelow = window.innerHeight - rect.bottom - GAP;
    const spaceAbove = rect.top - GAP;

    // Flip upward only when there is not enough room below but enough above
    const openAbove = spaceBelow < MENU_MAX_H && spaceAbove >= spaceBelow;

    setMenuStyle(
      openAbove
        ? {
            position: "fixed",
            bottom: window.innerHeight - rect.top + GAP,
            left: rect.left,
            width: rect.width,
            zIndex: 9999,
          }
        : {
            position: "fixed",
            top: rect.bottom + GAP,
            left: rect.left,
            width: rect.width,
            zIndex: 9999,
          },
    );

    setOpen(true);
  };

  return (
    <div className="relative">
      {/* ── Trigger ─────────────────────────────────────────────────────── */}
      <button
        ref={triggerRef}
        type="button"
        onClick={() => (open ? setOpen(false) : openMenu())}
        aria-haspopup="listbox"
        aria-expanded={open}
        className={cn(
          inputBase,
          "flex items-center justify-between gap-2 cursor-default",
          "pl-9 pr-3 text-left",
          open && "border-ring ring-2 ring-ring/20",
        )}
      >
        <span className="flex-1 truncate text-sm text-foreground">
          <span className="flex items-center gap-1.5">
            <span>{selected.emoji}</span>
            <span>{selected.name}</span>
          </span>
        </span>
        <ChevronDown
          size={13}
          strokeWidth={2.5}
          className={cn(
            "flex-shrink-0 text-muted-foreground/60 transition-transform duration-150",
            open && "rotate-180",
          )}
        />
      </button>

      {/* ── Dropdown — portalled into document.body so it is never      ── */}
      {/* ── clipped by DialogContent's overflow:hidden or trapped       ── */}
      {/* ── inside its CSS-animation transform containing block.        ── */}
      {mounted &&
        open &&
        createPortal(
          <>
            {/* Click-away backdrop */}
            <div
              className="fixed inset-0"
              style={{ zIndex: 9998 }}
              onClick={() => setOpen(false)}
              aria-hidden
            />

            <ul
              role="listbox"
              style={menuStyle}
              className={cn(
                "rounded-xl border border-border bg-popover",
                "shadow-xl shadow-black/[0.12]",
                "py-1 max-h-56 overflow-y-auto",
                "animate-in fade-in-0 zoom-in-95 duration-150 origin-top",
              )}
            >
              {CATEGORIES.map((cat) => {
                const isSelected = effectiveValue === cat.id;
                return (
                  <li
                    key={cat.id}
                    role="option"
                    aria-selected={isSelected}
                    onClick={() => {
                      onChange(cat.id);
                      setOpen(false);
                    }}
                    className={cn(
                      "flex items-center gap-2.5 px-3 py-2 text-sm cursor-default",
                      "hover:bg-accent hover:text-accent-foreground",
                      "transition-colors duration-75 select-none",
                      isSelected && "text-primary font-medium bg-accent/40",
                    )}
                  >
                    <span className="w-4 flex-shrink-0">
                      {isSelected && (
                        <Check
                          size={13}
                          className="text-primary"
                          strokeWidth={2.5}
                        />
                      )}
                    </span>
                    <span>{cat.emoji}</span>
                    <span>{cat.name}</span>
                  </li>
                );
              })}
            </ul>
          </>,
          document.body,
        )}
    </div>
  );
}
