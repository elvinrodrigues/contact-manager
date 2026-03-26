"use client";

import {
  Dialog,
  DialogContent,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Contact } from "@/lib/api-service";
import { ContactAvatar } from "@/components/contact-ui";
import { cn } from "@/lib/utils";
import { AlertTriangle, Phone, Mail, X } from "lucide-react";

interface DuplicateDialogProps {
  open: boolean;
  duplicates: Contact[];
  onCancel: () => void;
}

export function DuplicateDialog({
  open,
  duplicates,
  onCancel,
}: DuplicateDialogProps) {
  return (
    <Dialog open={open} onOpenChange={(v) => !v && onCancel()}>
      <DialogContent className="p-0 overflow-hidden gap-0 sm:max-w-[460px]">
        {/* ── Header ──────────────────────────────────────────────────────── */}
        <div className="px-6 pt-6 pb-5 border-b border-border">
          <div className="flex items-start gap-4">
            {/* Warning icon bubble */}
            <div className="flex-shrink-0 w-10 h-10 rounded-xl bg-amber-100 dark:bg-amber-900/30 flex items-center justify-center shadow-sm">
              <AlertTriangle
                size={18}
                className="text-amber-600 dark:text-amber-400"
                strokeWidth={2.2}
              />
            </div>

            {/* Title + description */}
            <div className="flex-1 min-w-0 pt-0.5">
              <DialogTitle className="text-base font-semibold text-foreground leading-snug">
                Duplicate Contact Detected
              </DialogTitle>
              <DialogDescription className="text-sm text-muted-foreground mt-1 leading-relaxed">
                {duplicates.length === 1
                  ? "A contact with the same phone number already exists in your list."
                  : `${duplicates.length} contacts with the same phone number already exist in your list.`}
              </DialogDescription>
            </div>

            {/* Close button */}
            <button
              onClick={onCancel}
              className={cn(
                "flex-shrink-0 w-7 h-7 rounded-lg flex items-center justify-center",
                "text-muted-foreground hover:text-foreground",
                "hover:bg-muted transition-colors duration-100",
                "-mt-0.5 -mr-1",
              )}
              aria-label="Close"
            >
              <X size={14} strokeWidth={2.5} />
            </button>
          </div>
        </div>

        {/* ── Duplicate contact cards ──────────────────────────────────────── */}
        <div
          className={cn(
            "px-6 py-4 space-y-2.5",
            duplicates.length > 3 && "max-h-64 overflow-y-auto",
          )}
        >
          {/* Section label */}
          <p className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70 mb-3">
            Existing contact{duplicates.length !== 1 ? "s" : ""}
          </p>

          {duplicates.map((contact) => (
            <div
              key={contact.id}
              className={cn(
                "flex items-start gap-3 rounded-xl p-3.5",
                "border border-border bg-muted/30",
                "transition-colors duration-100",
              )}
            >
              {/* Avatar */}
              <ContactAvatar name={contact.name} size="md" className="mt-0.5" />

              {/* Info */}
              <div className="flex-1 min-w-0 space-y-1">
                {/* Name */}
                <p className="text-sm font-semibold text-foreground leading-none truncate">
                  {contact.name}
                </p>

                {/* Phone */}
                <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
                  <Phone
                    size={11}
                    strokeWidth={2}
                    className="flex-shrink-0 text-muted-foreground/60"
                  />
                  <span className="font-mono tracking-tight">
                    {contact.phone}
                  </span>
                </p>

                {/* Email (if present) */}
                {contact.email && (
                  <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
                    <Mail
                      size={11}
                      strokeWidth={2}
                      className="flex-shrink-0 text-muted-foreground/60"
                    />
                    <span className="truncate">{contact.email}</span>
                  </p>
                )}
              </div>
            </div>
          ))}
        </div>

        {/* ── Footer ──────────────────────────────────────────────────────── */}
        <div className="px-6 py-4 border-t border-border bg-muted/30 flex flex-col gap-3">
          {/* Hint text */}
          <p className="text-xs text-muted-foreground leading-relaxed">
            To avoid duplicates, your contact was not saved. Review the existing
            entry above or use a different phone number.
          </p>

          {/* Actions */}
          <div className="flex justify-end">
            <Button
              variant="outline"
              size="sm"
              onClick={onCancel}
              className="h-9 px-5 rounded-xl text-sm font-medium"
            >
              Got it
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
