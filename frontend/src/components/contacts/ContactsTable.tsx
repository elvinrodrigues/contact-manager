import React, { useState } from "react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  MoreHorizontal,
  Pencil,
  Trash2,
  RotateCcw,
} from "lucide-react";
import { cn } from "@/lib/utils";
import type { Contact } from "@/types/contact";
import { Skeleton } from "@/components/ui/skeleton";
import { CategoryBadge } from "@/components/contacts/CategoryBadge";

interface ContactsTableProps {
  contacts: Contact[];
  isLoading?: boolean;
  isDeleted?: boolean;
  onEdit?: (contact: Contact) => void;
  onDelete?: (id: number) => void;
  onRestore?: (id: number) => void;
  onPermanentDelete?: (id: number) => void;
}

const AVATAR_COLORS = [
  "bg-blue-500/15 text-blue-700 dark:text-blue-300",
  "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300",
  "bg-violet-500/15 text-violet-700 dark:text-violet-300",
  "bg-amber-500/15 text-amber-700 dark:text-amber-300",
  "bg-rose-500/15 text-rose-700 dark:text-rose-300",
  "bg-cyan-500/15 text-cyan-700 dark:text-cyan-300",
  "bg-orange-500/15 text-orange-700 dark:text-orange-300",
];

function getInitials(name: string): string {
  const parts = name.trim().split(/\s+/);
  if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase();
  return name.slice(0, 2).toUpperCase();
}

function getAvatarColor(name: string): string {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }
  return AVATAR_COLORS[Math.abs(hash) % AVATAR_COLORS.length];
}

const ContactRow = React.memo(function ContactRow({
  contact,
  isDeleted,
  removingId,
  restoringId,
  onEdit,
  onDelete,
  onRestore,
  onPermanentDelete,
}: {
  contact: Contact;
  isDeleted?: boolean;
  removingId: number | null;
  restoringId: number | null;
  onEdit?: (contact: Contact) => void;
  onDelete?: (id: number) => void;
  onRestore?: (id: number) => void;
  onPermanentDelete?: (id: number) => void;
}) {
  const isRemoving = removingId === contact.id;
  const isRestoring = restoringId === contact.id;

  return (
    <TableRow
      className={cn(
        "group transition-all duration-250",
        isDeleted && "opacity-60",
        isRemoving && "animate-scale-out pointer-events-none",
        isRestoring && "animate-flash-success",
        !isRemoving && !isRestoring && "hover:bg-surface-muted/60"
      )}
    >
      {/* Avatar */}
      <TableCell className="w-12 py-3">
        <div
          className={cn(
            "flex h-8 w-8 items-center justify-center rounded-full text-xs font-semibold",
            getAvatarColor(contact.name)
          )}
        >
          {getInitials(contact.name)}
        </div>
      </TableCell>

      {/* Name + Phone */}
      <TableCell className="py-3">
        <div className="flex flex-col gap-0.5">
          <span className="text-sm font-medium">{contact.name}</span>
          <span className="text-xs text-muted-foreground">
            {contact.phone || "No phone"}
          </span>
          {isDeleted && contact.daysRemaining !== undefined && (
            <span className={cn(
              "text-[10px] font-medium leading-none max-w-max rounded-sm px-1.5 py-0.5 mt-0.5",
              contact.daysRemaining === 0 
                ? "bg-destructive/10 text-destructive" 
                : "bg-warning-muted text-warning"
            )}>
              {contact.daysRemaining === 0 
                ? "Deleting today" 
                : contact.daysRemaining === 1 
                  ? "1 day left" 
                  : `${contact.daysRemaining} days left`}
            </span>
          )}
        </div>
      </TableCell>

      {/* Email */}
      <TableCell className="hidden py-3 text-muted-foreground sm:table-cell">
        {contact.email || (
          <span className="text-muted-foreground/40">No email</span>
        )}
      </TableCell>

      {/* Category */}
      <TableCell className="hidden py-3 md:table-cell">
        <CategoryBadge categoryId={contact.categoryId} />
      </TableCell>

      {/* Actions */}
      <TableCell className="w-12 py-3 text-right">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8 opacity-0 transition-opacity duration-150 group-hover:opacity-100 data-[state=open]:opacity-100"
            >
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-40">
            {isDeleted ? (
              <>
                <DropdownMenuItem
                  onClick={() => onRestore?.(contact.id)}
                  className="text-success"
                >
                  <RotateCcw className="mr-2 h-3.5 w-3.5" />
                  Restore
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={() => onPermanentDelete?.(contact.id)}
                  className="text-destructive"
                >
                  <Trash2 className="mr-2 h-3.5 w-3.5" />
                  Delete forever
                </DropdownMenuItem>
              </>
            ) : (
              <>
                <DropdownMenuItem onClick={() => onEdit?.(contact)}>
                  <Pencil className="mr-2 h-3.5 w-3.5" />
                  Edit
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={() => onDelete?.(contact.id)}
                  className="text-destructive"
                >
                  <Trash2 className="mr-2 h-3.5 w-3.5" />
                  Delete
                </DropdownMenuItem>
              </>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      </TableCell>
    </TableRow>
  );
});

export function ContactsTable({
  contacts,
  isLoading,
  isDeleted,
  onEdit,
  onDelete,
  onRestore,
  onPermanentDelete,
}: ContactsTableProps) {
  const [removingId, setRemovingId] = useState<number | null>(null);
  const [restoringId, setRestoringId] = useState<number | null>(null);

  const handleDelete = async (id: number) => {
    setRemovingId(id);
    await new Promise((r) => setTimeout(r, 250));
    onDelete?.(id);
    setRemovingId(null);
  };

  const handlePermanentDelete = async (id: number) => {
    setRemovingId(id);
    await new Promise((r) => setTimeout(r, 250));
    onPermanentDelete?.(id);
    setRemovingId(null);
  };

  const handleRestore = async (id: number) => {
    setRestoringId(id);
    onRestore?.(id);
    await new Promise((r) => setTimeout(r, 600));
    setRestoringId(null);
  };

  if (isLoading) {
    return (
      <div className="space-y-2">
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-14 w-full rounded-lg" />
        ))}
      </div>
    );
  }

  if (!contacts.length) return null;

  return (
    <div className="overflow-hidden rounded-xl border border-border bg-card shadow-card">
      <Table>
        <TableHeader>
          <TableRow className="bg-surface-muted/50 hover:bg-surface-muted/50">
            <TableHead className="w-12" />
            <TableHead className="text-xs font-semibold uppercase tracking-wide">
              Name
            </TableHead>
            <TableHead className="hidden text-xs font-semibold uppercase tracking-wide sm:table-cell">
              Email
            </TableHead>
            <TableHead className="hidden text-xs font-semibold uppercase tracking-wide md:table-cell">
              Category
            </TableHead>
            <TableHead className="w-12" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {contacts.map((c) => (
            <ContactRow
              key={c.id}
              contact={c}
              isDeleted={isDeleted}
              removingId={removingId}
              restoringId={restoringId}
              onEdit={onEdit}
              onDelete={handleDelete}
              onRestore={handleRestore}
              onPermanentDelete={handlePermanentDelete}
            />
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
