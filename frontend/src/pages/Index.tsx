import { useState, useCallback, useMemo } from "react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Plus, Users, LayoutGrid } from "lucide-react";
import { SearchBar } from "@/components/contacts/SearchBar";
import { ContactsTable } from "@/components/contacts/ContactsTable";
import { PaginationControls } from "@/components/contacts/PaginationControls";
import { ContactFormModal } from "@/components/contacts/ContactFormModal";
import { DuplicateDialog, type DuplicateData } from "@/components/contacts/DuplicateDialog";
import {
  useContacts,
  useSearchContacts,
  useDeletedContacts,
  useCreateContact,
  useUpdateContact,
  useDeleteContact,
  useRestoreContact,
  usePermanentDeleteContact,
} from "@/hooks/use-contacts";
import type { Contact, ContactFormData } from "@/types/contact";
import { CATEGORIES } from "@/constants/categories";
import { Toggle } from "@/components/ui/toggle";

const LIMIT = 10;

export default function Index() {
  const [page, setPage] = useState(1);
  const [deletedPage, setDeletedPage] = useState(1);
  const [searchQuery, setSearchQuery] = useState("");
  const [formOpen, setFormOpen] = useState(false);
  const [editContact, setEditContact] = useState<Contact | null>(null);
  const [duplicateData, setDuplicateData] = useState<DuplicateData | null>(null);
  const [groupByCategory, setGroupByCategory] = useState(false);

  const isSearching = searchQuery.length >= 2;

  const contactsQuery = useContacts(page, LIMIT);
  const searchResults = useSearchContacts(searchQuery);
  const deletedQuery = useDeletedContacts(deletedPage, LIMIT);

  const createMutation = useCreateContact();
  const updateMutation = useUpdateContact();
  const deleteMutation = useDeleteContact();
  const restoreMutation = useRestoreContact();
  const permanentDeleteMutation = usePermanentDeleteContact();

  const handlePermanentDelete = (id: number) => {
    if (!confirm("Delete permanently? This cannot be undone.")) return;
    permanentDeleteMutation.mutate(id);
  };

  const handleSearch = useCallback((q: string) => setSearchQuery(q), []);

  const handleCreate = (data: ContactFormData) => {
    createMutation.mutate(data, {
      onSuccess: (res) => {
        if ((res.status === "duplicate" || res.status === "deleted_duplicate") && res.duplicates && res.incoming) {
          setDuplicateData({
            existing: res.duplicates[0],
            incoming: res.incoming,
            status: res.status,
          });
        } else {
          setFormOpen(false);
        }
      },
    });
  };

  const handleOverwrite = (merged: ContactFormData) => {
    if (!duplicateData) return;
    updateMutation.mutate(
      { id: duplicateData.existing.id, data: merged },
      { onSuccess: () => { setDuplicateData(null); setFormOpen(false); } }
    );
  };

  const handleRestore = (merged: ContactFormData) => {
    if (!duplicateData) return;
    restoreMutation.mutate(duplicateData.existing.id, {
      onSuccess: () => {
        updateMutation.mutate(
          { id: duplicateData.existing.id, data: merged },
          { onSuccess: () => { setDuplicateData(null); setFormOpen(false); } }
        );
      }
    });
  };

  const handleUpdate = (data: ContactFormData) => {
    if (!editContact) return;
    updateMutation.mutate(
      { id: editContact.id, data },
      { onSuccess: () => { setEditContact(null); setFormOpen(false); } }
    );
  };

  const openCreate = () => { setEditContact(null); setFormOpen(true); };
  const openEdit = (c: Contact) => { setEditContact(c); setFormOpen(true); };

  const activeContacts = isSearching
    ? searchResults.data?.contacts || []
    : contactsQuery.data?.contacts || [];

  const groupedContacts = useMemo(() => {
    if (!groupByCategory) return null;
    const groups: Record<string, Contact[]> = {};
    for (const cat of CATEGORIES) {
      groups[cat.name] = [];
    }
    groups["Uncategorized"] = [];
    for (const c of activeContacts) {
      const catName = CATEGORIES.find((cat) => cat.id === c.categoryId)?.name || "Uncategorized";
      if (!groups[catName]) groups[catName] = [];
      groups[catName].push(c);
    }
    return Object.entries(groups).filter(([, contacts]) => contacts.length > 0);
  }, [groupByCategory, activeContacts]);

  return (
    <div className="min-h-screen bg-background">
      <div className="mx-auto max-w-6xl px-4 py-8">
        <div className="mb-8 flex items-center gap-3">
          <Users className="h-8 w-8 text-primary" />
          <h1 className="text-3xl font-bold tracking-tight">Contacts</h1>
        </div>

        <Tabs defaultValue="active" className="space-y-6">
          <TabsList>
            <TabsTrigger value="active">Active</TabsTrigger>
            <TabsTrigger value="deleted">Deleted</TabsTrigger>
          </TabsList>

          <TabsContent value="active" className="space-y-4">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <SearchBar onSearch={handleSearch} />
              <div className="flex items-center gap-2">
                <Toggle
                  pressed={groupByCategory}
                  onPressedChange={setGroupByCategory}
                  aria-label="Group by category"
                  variant="outline"
                  size="sm"
                >
                  <LayoutGrid className="mr-1.5 h-4 w-4" />
                  Group
                </Toggle>
                <Button onClick={openCreate}>
                  <Plus className="mr-2 h-4 w-4" /> New Contact
                </Button>
              </div>
            </div>

            {(isSearching ? searchResults.isError : contactsQuery.isError) ? (
              <div className="flex items-center justify-center py-12 text-destructive">
                Failed to load contacts.
              </div>
            ) : groupByCategory && groupedContacts ? (
              <div className="space-y-6">
                {groupedContacts.map(([category, contacts]) => (
                  <div key={category}>
                    <h3 className="mb-2 text-sm font-semibold text-muted-foreground uppercase tracking-wider">
                      {category} ({contacts.length})
                    </h3>
                    <ContactsTable
                      contacts={contacts}
                      isLoading={isSearching ? searchResults.isLoading : contactsQuery.isLoading}
                      onEdit={openEdit}
                      onDelete={(id) => deleteMutation.mutate(id)}
                    />
                  </div>
                ))}
              </div>
            ) : (
              <ContactsTable
                contacts={activeContacts}
                isLoading={isSearching ? searchResults.isLoading : contactsQuery.isLoading}
                onEdit={openEdit}
                onDelete={(id) => deleteMutation.mutate(id)}
              />
            )}

            {!isSearching && contactsQuery.data && (
              <PaginationControls
                page={page}
                total={contactsQuery.data.total}
                limit={LIMIT}
                onPageChange={setPage}
              />
            )}
          </TabsContent>

          <TabsContent value="deleted" className="space-y-4">
            {deletedQuery.isError ? (
              <div className="flex items-center justify-center py-12 text-destructive">
                Failed to load deleted contacts.
              </div>
            ) : (
              <>
                <ContactsTable
                  contacts={deletedQuery.data?.contacts || []}
                  isLoading={deletedQuery.isLoading}
                  isDeleted
                  onRestore={(id) => restoreMutation.mutate(id)}
                  onPermanentDelete={handlePermanentDelete}
                />
                {deletedQuery.data && (
                  <PaginationControls
                    page={deletedPage}
                    total={deletedQuery.data.total}
                    limit={LIMIT}
                    onPageChange={setDeletedPage}
                  />
                )}
              </>
            )}
          </TabsContent>
        </Tabs>
      </div>

      <ContactFormModal
        open={formOpen}
        onClose={() => { setFormOpen(false); setEditContact(null); }}
        onSubmit={editContact ? handleUpdate : handleCreate}
        isSubmitting={createMutation.isPending || updateMutation.isPending}
        contact={editContact}
      />

      <DuplicateDialog
        data={duplicateData}
        onClose={() => setDuplicateData(null)}
        onOverwrite={handleOverwrite}
        onRestore={handleRestore}
      />
    </div>
  );
}
