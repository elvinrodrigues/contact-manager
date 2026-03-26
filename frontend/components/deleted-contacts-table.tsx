"use client";

import { useEffect, useState } from "react";
import { RotateCcw, Trash2, AlertCircle } from "lucide-react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { CategoryBadge, ContactAvatar } from "@/components/contact-ui";
import { toast } from "sonner";
import { cn } from "@/lib/utils";
import { Contact, getDeletedContacts, restoreContact } from "@/lib/api-service";

// ─── Skeleton rows ────────────────────────────────────────────────────────

function SkeletonRows() {
  return (
    <>
      {Array.from({ length: 6 }).map((_, i) => (
        <TableRow
          key={i}
          className={cn(
            "hover:bg-transparent border-0",
            i % 2 !== 0 ? "bg-muted/30" : "",
          )}
        >
          <TableCell className="pl-5 py-3">
            <div className="flex items-center gap-3">
              <Skeleton className="h-8 w-8 rounded-full flex-shrink-0" />
              <Skeleton className="h-4 w-28 rounded-md" />
            </div>
          </TableCell>
          <TableCell className="py-3">
            <Skeleton className="h-4 w-28 rounded-md" />
          </TableCell>
          <TableCell className="py-3">
            <Skeleton className="h-4 w-36 rounded-md" />
          </TableCell>
          <TableCell className="py-3">
            <Skeleton className="h-5 w-16 rounded-full" />
          </TableCell>
          <TableCell className="py-3">
            <Skeleton className="h-4 w-20 rounded-md" />
          </TableCell>
          <TableCell className="pr-5 py-3 text-right">
            <Skeleton className="h-7 w-20 rounded-lg ml-auto" />
          </TableCell>
        </TableRow>
      ))}
    </>
  );
}

// ─── Table header cell ────────────────────────────────────────────────────

function TH({
  children,
  className,
}: {
  children?: React.ReactNode;
  className?: string;
}) {
  return (
    <TableHead
      className={cn(
        "h-10 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/80 whitespace-nowrap select-none",
        className,
      )}
    >
      {children}
    </TableHead>
  );
}

// ─── Empty state ──────────────────────────────────────────────────────────

function EmptyDeletedState() {
  return (
    <div className="flex flex-col items-center justify-center py-16 px-4 text-center">
      {/* Icon bubble */}
      <div className="w-16 h-16 rounded-2xl bg-muted flex items-center justify-center mb-5 shadow-sm">
        <Trash2
          size={28}
          className="text-muted-foreground/50"
          strokeWidth={1.8}
        />
      </div>

      <h3 className="text-base font-semibold text-foreground mb-1.5">
        No deleted contacts
      </h3>

      <p className="text-sm text-muted-foreground max-w-xs leading-relaxed">
        Contacts you delete will appear here. You can restore them at any time.
      </p>
    </div>
  );
}

// ─── Error banner ─────────────────────────────────────────────────────────

function ErrorBanner({
  message,
  onRetry,
}: {
  message: string;
  onRetry: () => void;
}) {
  return (
    <div className="flex items-start gap-3 rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-3.5 text-sm text-destructive mb-4">
      <AlertCircle size={16} className="mt-0.5 flex-shrink-0" strokeWidth={2} />
      <div className="flex-1 min-w-0">
        <p className="font-medium leading-snug">{message}</p>
      </div>
      <button
        onClick={onRetry}
        className="text-xs font-semibold underline-offset-2 hover:underline flex-shrink-0 mt-0.5"
      >
        Retry
      </button>
    </div>
  );
}

// ─── Date formatter ───────────────────────────────────────────────────────

function formatDeletedDate(dateStr?: string | null): string {
  if (!dateStr) return "—";
  const d = new Date(dateStr);
  if (isNaN(d.getTime())) return "—";
  return d.toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

// ─── Props ────────────────────────────────────────────────────────────────

interface DeletedContactsTableProps {
  refreshTrigger?: number;
}

// ─── Main component ───────────────────────────────────────────────────────

export function DeletedContactsTable({
  refreshTrigger,
}: DeletedContactsTableProps) {
  const [deletedContacts, setDeletedContacts] = useState<Contact[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [restoringId, setRestoringId] = useState<number | null>(null);

  // Confirmation state
  const [confirmTarget, setConfirmTarget] = useState<Contact | null>(null);

  // ── Data loading ──────────────────────────────────────────────────────────

  const loadDeletedContacts = async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const data = await getDeletedContacts();
      setDeletedContacts(data);
    } catch (error) {
      console.error("Error fetching deleted contacts:", error);
      const msg =
        error instanceof Error
          ? error.message
          : "Failed to load deleted contacts";
      setLoadError(msg);
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  };

  // ── Restore ───────────────────────────────────────────────────────────────

  const handleRestoreRequest = (contact: Contact) => {
    setConfirmTarget(contact);
  };

  const handleRestoreConfirm = async () => {
    if (!confirmTarget) return;
    const id = confirmTarget.id;
    setRestoringId(id);
    try {
      await restoreContact(id);
      toast.success(`"${confirmTarget.name}" restored to active contacts`);
      setConfirmTarget(null);
      await loadDeletedContacts();
    } catch (error) {
      console.error("Error restoring contact:", error);
      toast.error(
        error instanceof Error ? error.message : "Failed to restore contact",
      );
    } finally {
      setRestoringId(null);
    }
  };

  // ── Effect ────────────────────────────────────────────────────────────────

  useEffect(() => {
    loadDeletedContacts();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [refreshTrigger]);

  // ── Derived state ─────────────────────────────────────────────────────────

  const isInitialLoad = loading && deletedContacts.length === 0;
  const isEmpty = !loading && !loadError && deletedContacts.length === 0;

  // ── Render ────────────────────────────────────────────────────────────────

  return (
    <>
      {/* ── Count label ─────────────────────────────────────────────────── */}
      {!loading && !loadError && deletedContacts.length > 0 && (
        <p className="text-xs text-muted-foreground mb-3 flex items-center gap-1.5">
          <Trash2 size={12} strokeWidth={2} />
          {deletedContacts.length} deleted contact
          {deletedContacts.length !== 1 ? "s" : ""}
        </p>
      )}

      {/* ── Error ─────────────────────────────────────────────────────────── */}
      {loadError && (
        <ErrorBanner message={loadError} onRetry={loadDeletedContacts} />
      )}

      {/* ── Empty state ──────────────────────────────────────────────────── */}
      {isEmpty && !loadError && (
        <div className="rounded-2xl border border-dashed border-border bg-card shadow-sm overflow-hidden">
          <EmptyDeletedState />
        </div>
      )}

      {/* ── Table ─────────────────────────────────────────────────────────── */}
      {(isInitialLoad || deletedContacts.length > 0) && !isEmpty && (
        <div className="rounded-2xl border border-border overflow-hidden bg-card shadow-sm">
          <Table>
            {/* Header */}
            <TableHeader>
              <TableRow className="bg-muted/40 hover:bg-muted/40 border-0 border-b border-border">
                <TH className="pl-5 w-[200px]">Name</TH>
                <TH className="w-[150px]">Phone</TH>
                <TH className="w-[200px]">Email</TH>
                <TH className="w-[110px]">Category</TH>
                <TH className="w-[130px]">Deleted on</TH>
                <TH className="pr-5 text-right w-[110px]">Actions</TH>
              </TableRow>
            </TableHeader>

            {/* Body */}
            <TableBody>
              {isInitialLoad ? (
                <SkeletonRows />
              ) : (
                deletedContacts.map((contact, idx) => {
                  const isEvenRow = idx % 2 !== 0;
                  const isBeingRestored = restoringId === contact.id;

                  return (
                    <TableRow
                      key={contact.id}
                      className={cn(
                        "group border-0 transition-colors duration-100",
                        isEvenRow
                          ? "bg-muted/25 hover:bg-muted/50"
                          : "hover:bg-muted/30",
                        isBeingRestored && "opacity-50",
                      )}
                    >
                      {/* Name + avatar */}
                      <TableCell className="pl-5 py-3">
                        <div className="flex items-center gap-3">
                          <ContactAvatar name={contact.name} />
                          <span className="text-sm font-medium text-foreground truncate max-w-[130px]">
                            {contact.name}
                          </span>
                        </div>
                      </TableCell>

                      {/* Phone */}
                      <TableCell className="py-3 text-sm text-muted-foreground font-mono tracking-tight">
                        {contact.phone}
                      </TableCell>

                      {/* Email */}
                      <TableCell className="py-3 text-sm">
                        {contact.email ? (
                          <span className="text-muted-foreground truncate max-w-[180px] inline-block">
                            {contact.email}
                          </span>
                        ) : (
                          <span className="text-muted-foreground/40 italic text-xs">
                            —
                          </span>
                        )}
                      </TableCell>

                      {/* Category */}
                      <TableCell className="py-3">
                        <CategoryBadge categoryId={contact.category_id} />
                      </TableCell>

                      {/* Deleted date */}
                      <TableCell className="py-3 text-sm text-muted-foreground tabular-nums">
                        {formatDeletedDate(contact.deleted_at)}
                      </TableCell>

                      {/* Restore action */}
                      <TableCell className="pr-5 py-3 text-right">
                        <button
                          onClick={() => handleRestoreRequest(contact)}
                          disabled={isBeingRestored}
                          title="Restore contact"
                          className={cn(
                            "inline-flex items-center gap-1.5 h-7 px-2.5 rounded-lg text-xs font-medium",
                            "text-emerald-700 dark:text-emerald-400",
                            "bg-emerald-50 dark:bg-emerald-900/25",
                            "border border-emerald-200 dark:border-emerald-800/60",
                            "hover:bg-emerald-100 dark:hover:bg-emerald-900/40",
                            "active:scale-95 transition-all duration-100",
                            "opacity-0 group-hover:opacity-100 focus-visible:opacity-100",
                            "disabled:pointer-events-none disabled:opacity-40",
                          )}
                        >
                          {isBeingRestored ? (
                            <>
                              <Spinner className="h-3 w-3" />
                              <span>Restoring…</span>
                            </>
                          ) : (
                            <>
                              <RotateCcw size={11} strokeWidth={2.5} />
                              <span>Restore</span>
                            </>
                          )}
                        </button>
                      </TableCell>
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>
        </div>
      )}

      {/* ── Restore confirmation dialog ───────────────────────────────────── */}
      <ConfirmDialog
        open={confirmTarget !== null}
        onOpenChange={(open) => {
          if (!open) setConfirmTarget(null);
        }}
        title="Restore this contact?"
        description={
          confirmTarget
            ? `"${confirmTarget.name}" will be moved back to your active contacts list and will appear in search results again.`
            : ""
        }
        confirmLabel="Restore contact"
        cancelLabel="Cancel"
        onConfirm={handleRestoreConfirm}
        isLoading={restoringId !== null}
        variant="default"
      />
    </>
  );
}
