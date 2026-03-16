"use client";

import { useEffect, useState, useRef } from "react";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Empty } from "@/components/ui/empty";
import { Spinner } from "@/components/ui/spinner";
import { toast } from "sonner";
import {
  Contact,
  getContacts,
  searchContacts,
  deleteContact,
} from "@/lib/api-service";

import {
  EmptyHeader,
  EmptyTitle,
  EmptyDescription,
} from "@/components/ui/empty";

interface ContactsTableProps {
  refreshTrigger?: number;
}

export function ContactsTable({ refreshTrigger }: ContactsTableProps) {
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState("");
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [groupByCategory, setGroupByCategory] = useState(false);

  const debounceTimer = useRef<NodeJS.Timeout | null>(null);

  // Group contacts by category
  const groupedContacts = (contacts: Contact[]) => {
    const groups: { [key: string]: Contact[] } = {};

    contacts.forEach((contact) => {
      const category = contact.category_id?.toString() || "Uncategorized";

      if (!groups[category]) {
        groups[category] = [];
      }

      groups[category].push(contact);
    });

    return Object.keys(groups)
      .sort()
      .reduce(
        (acc, key) => {
          acc[key] = groups[key];
          return acc;
        },
        {} as { [key: string]: Contact[] },
      );
  };

  // Load contacts
  const loadContacts = async () => {
    setLoading(true);

    try {
      const data = await getContacts();
      setContacts(data);
    } catch (error) {
      console.error("Error fetching contacts:", error);
      toast.error(
        error instanceof Error ? error.message : "Failed to load contacts",
      );
    } finally {
      setLoading(false);
    }
  };

  // Search with debounce
  const handleSearch = (query: string) => {
    setSearchQuery(query);

    if (debounceTimer.current) {
      clearTimeout(debounceTimer.current);
    }

    if (!query.trim()) {
      debounceTimer.current = setTimeout(() => {
        loadContacts();
      }, 400);

      return;
    }

    debounceTimer.current = setTimeout(async () => {
      setLoading(true);

      try {
        const data = await searchContacts(query);
        setContacts(data);
      } catch (error) {
        console.error("Error searching contacts:", error);
        toast.error(
          error instanceof Error ? error.message : "Failed to search contacts",
        );
      } finally {
        setLoading(false);
      }
    }, 400);
  };

  // Delete contact
  const handleDelete = async (id: number) => {
    setDeletingId(id);

    try {
      await deleteContact(id);

      toast.success("Contact deleted successfully");

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

  useEffect(() => {
    loadContacts();
  }, [refreshTrigger]);

  if (loading && contacts.length === 0) {
    return (
      <div className="space-y-4">
        <Input placeholder="Search contacts..." disabled className="max-w-md" />

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
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <Input
          placeholder="Search contacts..."
          value={searchQuery}
          onChange={(e) => handleSearch(e.target.value)}
          className="max-w-md"
        />

        <Button
          variant={groupByCategory ? "default" : "outline"}
          onClick={() => setGroupByCategory(!groupByCategory)}
          className="w-full sm:w-auto"
        >
          {groupByCategory ? "Grouped by Category" : "Group by Category"}
        </Button>
      </div>

      {contacts.length === 0 ? (
        <Empty>
          <EmptyHeader>
            <EmptyTitle>
              {searchQuery ? "No search results" : "No contacts"}
            </EmptyTitle>

            <EmptyDescription>
              {searchQuery
                ? "Try a different search term"
                : "Start by adding your first contact"}
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : groupByCategory ? (
        <div className="space-y-6">
          {Object.entries(groupedContacts(contacts)).map(
            ([category, categoryContacts]) => (
              <div key={category}>
                <h3 className="text-lg font-semibold mb-3">{category}</h3>

                <div className="border rounded-lg overflow-hidden">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Name</TableHead>
                        <TableHead>Phone</TableHead>
                        <TableHead>Email</TableHead>
                        <TableHead className="w-20">Actions</TableHead>
                      </TableRow>
                    </TableHeader>

                    <TableBody>
                      {categoryContacts.map((contact) => (
                        <TableRow key={contact.id}>
                          <TableCell>{contact.name}</TableCell>
                          <TableCell>{contact.phone}</TableCell>
                          <TableCell>{contact.email}</TableCell>

                          <TableCell>
                            <Button
                              variant="destructive"
                              size="sm"
                              disabled={deletingId === contact.id}
                              onClick={() => handleDelete(contact.id)}
                            >
                              {deletingId === contact.id ? (
                                <Spinner className="h-4 w-4" />
                              ) : (
                                "Delete"
                              )}
                            </Button>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              </div>
            ),
          )}
        </div>
      ) : (
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
              {contacts.map((contact) => (
                <TableRow key={contact.id}>
                  <TableCell>{contact.name}</TableCell>
                  <TableCell>{contact.phone}</TableCell>
                  <TableCell>{contact.email}</TableCell>
                  <TableCell>{contact.category_id}</TableCell>

                  <TableCell>
                    <Button
                      variant="destructive"
                      size="sm"
                      disabled={deletingId === contact.id}
                      onClick={() => handleDelete(contact.id)}
                    >
                      {deletingId === contact.id ? (
                        <Spinner className="h-4 w-4" />
                      ) : (
                        "Delete"
                      )}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
