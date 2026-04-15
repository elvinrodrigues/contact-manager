import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Pencil, Trash2, RotateCcw } from "lucide-react";
import type { Contact } from "@/types/contact";
import { Skeleton } from "@/components/ui/skeleton";

import { CATEGORIES } from "@/constants/categories";

function getCategoryName(id?: number) {
  return CATEGORIES.find((c) => c.id === id)?.name || "—";
}

interface ContactsTableProps {
  contacts: Contact[];
  isLoading?: boolean;
  isDeleted?: boolean;
  onEdit?: (contact: Contact) => void;
  onDelete?: (id: number) => void;
  onRestore?: (id: number) => void;
}

export function ContactsTable({
  contacts,
  isLoading,
  isDeleted,
  onEdit,
  onDelete,
  onRestore,
}: ContactsTableProps) {
  if (isLoading) {
    return (
      <div className="space-y-2">
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-12 w-full" />
        ))}
      </div>
    );
  }

  if (!contacts.length) {
    return (
      <div className="flex items-center justify-center py-12 text-muted-foreground">
        No contacts found.
      </div>
    );
  }

  return (
    <div className="rounded-lg border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Phone</TableHead>
            <TableHead className="hidden sm:table-cell">Email</TableHead>
            <TableHead className="hidden md:table-cell">Category</TableHead>
            <TableHead className="text-right">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {contacts.map((c) => (
            <TableRow key={c.id}>
              <TableCell className="font-medium">{c.name}</TableCell>
              <TableCell>{c.phone}</TableCell>
              <TableCell className="hidden sm:table-cell">
                {c.email || "—"}
              </TableCell>
              <TableCell className="hidden md:table-cell">
                {getCategoryName(c.categoryId)}
              </TableCell>
              <TableCell className="text-right">
                {isDeleted ? (
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => onRestore?.(c.id)}
                  >
                    <RotateCcw className="h-4 w-4" />
                  </Button>
                ) : (
                  <div className="flex justify-end gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => onEdit?.(c)}
                    >
                      <Pencil className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => onDelete?.(c.id)}
                    >
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </div>
                )}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
