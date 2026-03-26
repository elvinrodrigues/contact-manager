"use client";

import { useEffect, useState, useRef } from "react";
import {
  Search,
  X,
  Users,
  LayoutGrid,
  List,
  Trash2,
  AlertCircle,
} from "lucide-react";
import { Button } from "@/components/ui/button";
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
import {
  Contact,
  getContacts,
  searchContacts,
  deleteContact,
} from "@/lib/api-service";

// ─── Skeleton loading rows ────────────────────────────────────────────────

function SkeletonRows({ cols = 5 }: { cols?: number }) {
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
          {cols >= 4 && (
            <TableCell className="py-3">
              <Skeleton className="h-4 w-36 rounded-md" />
            </TableCell>
          )}
          {cols >= 5 && (
            <TableCell className="py-3">
              <Skeleton className="h-5 w-16 rounded-full" />
            </TableCell>
          )}
          <TableCell className="pr-5 py-3 text-right">
            <Skeleton className="h-7 w-7 rounded-lg ml-auto" />
          </TableCell>
        </TableRow>
      ))}
    </>
  );
}

// ─── Compact table header cell ────────────────────────────────────────────

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

function EmptyContactsState({
  isSearch,
  query,
}: {
  isSearch: boolean;
  query: string;
}) {
  return (
    <div className="flex flex-col items-center justify-center py-16 px-4 text-center">
      {/* Icon bubble */}
      <div className="w-16 h-16 rounded-2xl bg-primary/10 flex items-center justify-center mb-5 shadow-sm">
        {isSearch ? (
          <Search size={28} className="text-primary/70" strokeWidth={1.8} />
        ) : (
          <Users size={28} className="text-primary/70" strokeWidth={1.8} />
        )}
      </div>

      <h3 className="text-base font-semibold text-foreground mb-1.5">
        {isSearch ? "No results found" : "No contacts yet"}
      </h3>

      <p className="text-sm text-muted-foreground max-w-xs leading-relaxed">
        {isSearch
          ? `No contacts matched "${query}". Try adjusting your search.`
          : "Add your first contact using the button above to get started."}
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

// ─── Main component ───────────────────────────────────────────────────────

interface ContactsTableProps {
  refreshTrigger?: number;
}

export function ContactsTable({ refreshTrigger }: ContactsTableProps) {
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [groupByCategory, setGroupByCategory] = useState(false);

  // Delete confirmation state
  const [confirmTarget, setConfirmTarget] = useState<Contact | null>(null);

  const debounceTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);

  // ── Data loading ─────────────────────────────────────────────────────────

  const loadContacts = async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const data = await getContacts();
      setContacts(data);
    } catch (error) {
      console.error("Error fetching contacts:", error);
      const msg =
        error instanceof Error ? error.message : "Failed to load contacts";
      setLoadError(msg);
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  };

  // ── Search (debounced) ────────────────────────────────────────────────────

  const handleSearch = (query: string) => {
    setSearchQuery(query);
    if (debounceTimer.current) clearTimeout(debounceTimer.current);

    if (!query.trim()) {
      debounceTimer.current = setTimeout(() => loadContacts(), 400);
      return;
    }

    debounceTimer.current = setTimeout(async () => {
      setLoading(true);
      setLoadError(null);
      try {
        const data = await searchContacts(query);
        setContacts(data);
      } catch (error) {
        console.error("Error searching contacts:", error);
        const msg =
          error instanceof Error ? error.message : "Failed to search contacts";
        setLoadError(msg);
        toast.error(msg);
      } finally {
        setLoading(false);
      }
    }, 400);
  };

  const clearSearch = () => {
    handleSearch("");
    searchInputRef.current?.focus();
  };

  // ── Delete ────────────────────────────────────────────────────────────────

  const handleDeleteRequest = (contact: Contact) => {
    setConfirmTarget(contact);
  };

  const handleDeleteConfirm = async () => {
    if (!confirmTarget) return;
    const id = confirmTarget.id;
    setDeletingId(id);

    try {
      await deleteContact(id);
      toast.success(`"${confirmTarget.name}" deleted`);
      setConfirmTarget(null);
      await loadContacts();
    } catch (error) {
      console.error("Error deleting contact:", error);
      toast.error(
        error instanceof Error ? error.message : "Failed to delete contact",
      );
    } finally {
      setDeletingId(null);
    }
  };

  // ── Group by category ─────────────────────────────────────────────────────

  const groupedContacts = (list: Contact[]) => {
    const groups: Record<string, Contact[]> = {};
    list.forEach((c) => {
      const key = c.category_id?.toString() ?? "0";
      if (!groups[key]) groups[key] = [];
      groups[key].push(c);
    });
    return Object.entries(groups).sort(([a], [b]) => a.localeCompare(b));
  };

  // ── Effect ────────────────────────────────────────────────────────────────

  useEffect(() => {
    loadContacts();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [refreshTrigger]);

  // ── Render helpers ────────────────────────────────────────────────────────

  const isInitialLoad = loading && contacts.length === 0;
  const isEmpty = !loading && !loadError && contacts.length === 0;
  const isSearch = searchQuery.trim().length > 0;

  // ── Shared table structure ─────────────────────────────────────────────

  function renderTableRows(list: Contact[], startIdx = 0) {
    return list.map((contact, idx) => {
      const isEvenRow = (startIdx + idx) % 2 !== 0;
      const isBeingDeleted = deletingId === contact.id;

      return (
        <TableRow
          key={contact.id}
          className={cn(
            "group border-0 transition-colors duration-100",
            isEvenRow ? "bg-muted/25 hover:bg-muted/50" : "hover:bg-muted/30",
            isBeingDeleted && "opacity-50",
          )}
        >
          {/* Name + avatar */}
          <TableCell className="pl-5 py-3 font-medium">
            <div className="flex items-center gap-3">
              <ContactAvatar name={contact.name} />
              <span className="text-foreground text-sm font-medium truncate max-w-[160px]">
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
              <a
                href={`mailto:${contact.email}`}
                className="text-primary hover:underline underline-offset-2 truncate max-w-[200px] inline-block"
              >
                {contact.email}
              </a>
            ) : (
              <span className="text-muted-foreground/40 italic text-xs">—</span>
            )}
          </TableCell>

          {/* Category */}
          <TableCell className="py-3">
            <CategoryBadge categoryId={contact.category_id} />
          </TableCell>

          {/* Actions */}
          <TableCell className="pr-5 py-3 text-right">
            <button
              onClick={() => handleDeleteRequest(contact)}
              disabled={isBeingDeleted}
              title="Delete contact"
              className={cn(
                "inline-flex items-center justify-center w-7 h-7 rounded-lg",
                "text-muted-foreground/50 hover:text-destructive",
                "hover:bg-destructive/10 active:scale-95",
                "transition-all duration-100",
                "opacity-0 group-hover:opacity-100 focus-visible:opacity-100",
                "disabled:pointer-events-none disabled:opacity-30",
              )}
            >
              {isBeingDeleted ? (
                <Spinner className="h-3.5 w-3.5" />
              ) : (
                <Trash2 size={14} strokeWidth={2} />
              )}
            </button>
          </TableCell>
        </TableRow>
      );
    });
  }

  // ─────────────────────────────────────────────────────────────────────────

  return (
    <>
      {/* ── Toolbar ─────────────────────────────────────────────────────── */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between mb-5">
        {/* Search input */}
        <div className="relative w-full sm:max-w-sm">
          <Search
            size={15}
            className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground pointer-events-none"
            strokeWidth={2}
          />
          <input
            ref={searchInputRef}
            type="search"
            placeholder="Search by name, phone, or email…"
            value={searchQuery}
            onChange={(e) => handleSearch(e.target.value)}
            className={cn(
              "w-full h-9 pl-9 pr-9 rounded-xl text-sm",
              "bg-card border border-border",
              "text-foreground placeholder:text-muted-foreground/70",
              "shadow-sm",
              "transition-[border-color,box-shadow] duration-150",
              "focus:outline-none focus:border-ring focus:ring-2 focus:ring-ring/25",
              "disabled:opacity-50",
            )}
          />
          {searchQuery && (
            <button
              onClick={clearSearch}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors rounded"
              aria-label="Clear search"
            >
              <X size={13} strokeWidth={2.5} />
            </button>
          )}
        </div>

        {/* Right controls */}
        <div className="flex items-center gap-2 flex-shrink-0">
          {/* Result count pill */}
          {!loading && contacts.length > 0 && (
            <span className="text-xs text-muted-foreground bg-muted border border-border rounded-full px-2.5 py-1 font-medium hidden sm:inline-flex items-center gap-1">
              <Users size={11} />
              {contacts.length} {isSearch ? "result" : "contact"}
              {contacts.length !== 1 ? "s" : ""}
            </span>
          )}

          {/* Group by toggle */}
          <Button
            variant={groupByCategory ? "secondary" : "outline"}
            size="sm"
            onClick={() => setGroupByCategory((v) => !v)}
            disabled={loading || contacts.length === 0}
            className={cn(
              "h-9 px-3 gap-1.5 text-xs font-medium rounded-xl transition-colors",
              groupByCategory &&
                "bg-accent text-accent-foreground border-primary/30",
            )}
          >
            {groupByCategory ? (
              <>
                <List size={13} strokeWidth={2} />
                <span className="hidden sm:inline">List view</span>
                <span className="sm:hidden">List</span>
              </>
            ) : (
              <>
                <LayoutGrid size={13} strokeWidth={2} />
                <span className="hidden sm:inline">Group by category</span>
                <span className="sm:hidden">Group</span>
              </>
            )}
          </Button>
        </div>
      </div>

      {/* Mobile count */}
      {!loading && contacts.length > 0 && (
        <p className="text-xs text-muted-foreground mb-3 sm:hidden">
          {contacts.length} {isSearch ? "result" : "contact"}
          {contacts.length !== 1 ? "s" : ""}
        </p>
      )}

      {/* ── Error ───────────────────────────────────────────────────────── */}
      {loadError && <ErrorBanner message={loadError} onRetry={loadContacts} />}

      {/* ── Empty state ──────────────────────────────────────────────────── */}
      {isEmpty && !loadError && (
        <div className="rounded-2xl border border-dashed border-border bg-card shadow-sm overflow-hidden">
          <EmptyContactsState isSearch={isSearch} query={searchQuery} />
        </div>
      )}

      {/* ── Grouped view ─────────────────────────────────────────────────── */}
      {(loading || contacts.length > 0) && groupByCategory && !isEmpty && (
        <div className="space-y-5">
          {isInitialLoad
            ? // Skeleton groups
              [1, 2].map((k) => (
                <div key={k}>
                  <div className="flex items-center gap-2 mb-3 px-1">
                    <Skeleton className="h-4 w-16 rounded-md" />
                    <Skeleton className="h-4 w-6 rounded-full" />
                  </div>
                  <div className="rounded-2xl border border-border overflow-hidden bg-card shadow-sm">
                    <Table>
                      <TableHeader>
                        <TableRow className="bg-muted/40 hover:bg-muted/40 border-0">
                          <TH className="pl-5 w-[220px]">Name</TH>
                          <TH className="w-[160px]">Phone</TH>
                          <TH>Email</TH>
                          <TH className="pr-5 text-right w-12" />
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <SkeletonRows cols={4} />
                      </TableBody>
                    </Table>
                  </div>
                </div>
              ))
            : groupedContacts(contacts).map(([categoryId, group]) => (
                <div key={categoryId}>
                  {/* Group header */}
                  <div className="flex items-center gap-2 mb-3 px-1">
                    <CategoryBadge
                      categoryId={parseInt(categoryId)}
                      className="text-[11px] px-2.5 py-0.5"
                    />
                    <span className="inline-flex items-center rounded-full bg-muted border border-border px-2 py-0.5 text-[11px] font-medium text-muted-foreground">
                      {group.length}
                    </span>
                  </div>

                  {/* Group table */}
                  <div className="rounded-2xl border border-border overflow-hidden bg-card shadow-sm">
                    <Table>
                      <TableHeader>
                        <TableRow className="bg-muted/40 hover:bg-muted/40 border-0 border-b border-border">
                          <TH className="pl-5 w-[220px]">Name</TH>
                          <TH className="w-[160px]">Phone</TH>
                          <TH>Email</TH>
                          <TH className="pr-5 text-right w-12" />
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {group.map((contact, idx) => {
                          const isEvenRow = idx % 2 !== 0;
                          const isBeingDeleted = deletingId === contact.id;
                          return (
                            <TableRow
                              key={contact.id}
                              className={cn(
                                "group border-0 transition-colors duration-100",
                                isEvenRow
                                  ? "bg-muted/25 hover:bg-muted/50"
                                  : "hover:bg-muted/30",
                                isBeingDeleted && "opacity-50",
                              )}
                            >
                              <TableCell className="pl-5 py-3 font-medium">
                                <div className="flex items-center gap-3">
                                  <ContactAvatar name={contact.name} />
                                  <span className="text-sm font-medium truncate max-w-[140px]">
                                    {contact.name}
                                  </span>
                                </div>
                              </TableCell>
                              <TableCell className="py-3 text-sm text-muted-foreground font-mono tracking-tight">
                                {contact.phone}
                              </TableCell>
                              <TableCell className="py-3 text-sm">
                                {contact.email ? (
                                  <a
                                    href={`mailto:${contact.email}`}
                                    className="text-primary hover:underline underline-offset-2"
                                  >
                                    {contact.email}
                                  </a>
                                ) : (
                                  <span className="text-muted-foreground/40 italic text-xs">
                                    —
                                  </span>
                                )}
                              </TableCell>
                              <TableCell className="pr-5 py-3 text-right">
                                <button
                                  onClick={() => handleDeleteRequest(contact)}
                                  disabled={isBeingDeleted}
                                  title="Delete contact"
                                  className={cn(
                                    "inline-flex items-center justify-center w-7 h-7 rounded-lg",
                                    "text-muted-foreground/50 hover:text-destructive",
                                    "hover:bg-destructive/10 active:scale-95",
                                    "transition-all duration-100",
                                    "opacity-0 group-hover:opacity-100 focus-visible:opacity-100",
                                    "disabled:pointer-events-none",
                                  )}
                                >
                                  {isBeingDeleted ? (
                                    <Spinner className="h-3.5 w-3.5" />
                                  ) : (
                                    <Trash2 size={14} strokeWidth={2} />
                                  )}
                                </button>
                              </TableCell>
                            </TableRow>
                          );
                        })}
                      </TableBody>
                    </Table>
                  </div>
                </div>
              ))}
        </div>
      )}

      {/* ── Flat list view ────────────────────────────────────────────────── */}
      {(isInitialLoad || contacts.length > 0) &&
        !groupByCategory &&
        !isEmpty && (
          <div className="rounded-2xl border border-border overflow-hidden bg-card shadow-sm">
            <Table>
              <TableHeader>
                <TableRow className="bg-muted/40 hover:bg-muted/40 border-0 border-b border-border">
                  <TH className="pl-5 w-[220px]">Name</TH>
                  <TH className="w-[160px]">Phone</TH>
                  <TH className="w-[200px]">Email</TH>
                  <TH className="w-[110px]">Category</TH>
                  <TH className="pr-5 text-right w-12" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {isInitialLoad ? (
                  <SkeletonRows cols={5} />
                ) : (
                  renderTableRows(contacts)
                )}
              </TableBody>
            </Table>
          </div>
        )}

      {/* ── Delete confirmation dialog ────────────────────────────────────── */}
      <ConfirmDialog
        open={confirmTarget !== null}
        onOpenChange={(open) => {
          if (!open) setConfirmTarget(null);
        }}
        title="Delete this contact?"
        description={
          confirmTarget
            ? `"${confirmTarget.name}" will be moved to the deleted contacts list. You can restore them at any time from the Deleted tab.`
            : ""
        }
        confirmLabel="Delete contact"
        cancelLabel="Keep contact"
        onConfirm={handleDeleteConfirm}
        isLoading={deletingId !== null}
        variant="destructive"
      />
    </>
  );
}
