"use client";

import { useEffect, useState } from "react";
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
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from "@/components/ui/empty";
import { Spinner } from "@/components/ui/spinner";
import { toast } from "sonner";
import { Contact, getDeletedContacts, restoreContact } from "@/lib/api-service";

interface DeletedContactsTableProps {
  refreshTrigger?: number;
}

export function DeletedContactsTable({
  refreshTrigger,
}: DeletedContactsTableProps) {
  const [deletedContacts, setDeletedContacts] = useState<Contact[]>([]);
  const [loading, setLoading] = useState(true);
  const [restoringId, setRestoringId] = useState<Number | null>(null);

  // Load deleted contacts
  const loadDeletedContacts = async () => {
    setLoading(true);
    try {
      const data = await getDeletedContacts();
      setDeletedContacts(data);
    } catch (error) {
      console.error("Error fetching deleted contacts:", error);
      toast.error(
        error instanceof Error
          ? error.message
          : "Failed to load deleted contacts",
      );
    } finally {
      setLoading(false);
    }
  };

  // Restore contact
  const handleRestore = async (id: number) => {
    setRestoringId(Number(id));
    try {
      await restoreContact(Number(id));
      toast.success("Contact restored successfully");
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

  // Load on mount and when refreshTrigger changes
  useEffect(() => {
    loadDeletedContacts();
  }, [refreshTrigger]);

  if (loading && deletedContacts.length === 0) {
    return (
      <div className="border rounded-lg overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Phone</TableHead>
              <TableHead>Email</TableHead>
              <TableHead>Category</TableHead>
              <TableHead className="w-20">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {Array.from({ length: 5 }).map((_, i) => (
              <TableRow key={i}>
                <TableCell>
                  <Skeleton className="h-4 w-24" />
                </TableCell>
                <TableCell>
                  <Skeleton className="h-4 w-32" />
                </TableCell>
                <TableCell>
                  <Skeleton className="h-4 w-40" />
                </TableCell>
                <TableCell>
                  <Skeleton className="h-4 w-16" />
                </TableCell>
                <TableCell>
                  <Skeleton className="h-4 w-16" />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    );
  }

  if (deletedContacts.length === 0) {
    return (
      <Empty>
        <EmptyHeader>
          <EmptyTitle>No deleted contacts</EmptyTitle>
          <EmptyDescription>
            You haven't deleted any contacts yet
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    );
  }

  return (
    <div className="border rounded-lg overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Phone</TableHead>
            <TableHead>Email</TableHead>
            <TableHead>Category</TableHead>
            <TableHead className="w-20">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {deletedContacts.map((contact) => (
            <TableRow key={contact.id}>
              <TableCell>{contact.name}</TableCell>
              <TableCell>{contact.phone}</TableCell>
              <TableCell>{contact.email}</TableCell>
              <TableCell>{contact.category_id}</TableCell>
              <TableCell>
                <Button
                  variant="default"
                  size="sm"
                  disabled={restoringId === contact.id}
                  onClick={() => handleRestore(contact.id)}
                >
                  {restoringId === contact.id ? (
                    <Spinner className="h-4 w-4" />
                  ) : (
                    "Restore"
                  )}
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
