import { useState, useCallback, useMemo } from "react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Plus, InfoIcon } from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { SearchBar } from "@/components/contacts/SearchBar";
import { ContactsTable } from "@/components/contacts/ContactsTable";
import { ContactsEmptyState } from "@/components/contacts/ContactsEmptyState";
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


const LIMIT = 10;

export default function Index() {
  const [page, setPage] = useState(1);
  const [deletedPage, setDeletedPage] = useState(1);
  const [searchQuery, setSearchQuery] = useState("");
  const [formOpen, setFormOpen] = useState(false);
  const [editContact, setEditContact] = useState<Contact | null>(null);
  const [duplicateData, setDuplicateData] = useState<DuplicateData | null>(null);
  const [categoryFilter, setCategoryFilter] = useState<string>("all");
  const [activeTab, setActiveTab] = useState("active");

  const isSearching = searchQuery.length >= 2;

  const contactsQuery = useContacts(page, LIMIT, categoryFilter);
  const searchResults = useSearchContacts(searchQuery);
  const deletedQuery = useDeletedContacts(deletedPage, LIMIT);

  const createMutation = useCreateContact();
  const updateMutation = useUpdateContact();
  const deleteMutation = useDeleteContact();
  const restoreMutation = useRestoreContact();
  const permanentDeleteMutation = usePermanentDeleteContact();

  const handlePermanentDelete = (id: number) => {
    if (!confirm("Are you sure? This cannot be undone")) return;
    permanentDeleteMutation.mutate(id);
  };

  const handleDelete = (id: number) => {
    if (!confirm("Are you sure? This cannot be undone")) return;
    deleteMutation.mutate(id);
  };

  const handleSearch = useCallback((q: string) => {
    setSearchQuery(q);
    setPage(1);
  }, []);

  const handleClearSearch = useCallback(() => {
    setSearchQuery("");
    setPage(1);
  }, []);

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

  const handleCategoryChange = (val: string) => {
    setCategoryFilter(val);
    setPage(1);
  };

  const handleTabChange = (value: string) => {
    setActiveTab(value);
    setSearchQuery("");
    setPage(1);
    setDeletedPage(1);
  };

  const totalActive = contactsQuery.data?.total ?? 0;
  const totalDeleted = deletedQuery.data?.total ?? 0;

  return (
    <div className="animate-fade-in">
      <Tabs value={activeTab} onValueChange={handleTabChange} className="space-y-5">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <TabsList>
            <TabsTrigger value="active">
              Active
              {totalActive > 0 && (
                <span className="ml-1.5 text-xs text-muted-foreground">({totalActive})</span>
              )}
            </TabsTrigger>
            <TabsTrigger value="deleted">
              Archived
              {totalDeleted > 0 && (
                <span className="ml-1.5 text-xs text-muted-foreground">({totalDeleted})</span>
              )}
            </TabsTrigger>
          </TabsList>

          {activeTab === "active" && (
            <div className="flex items-center gap-2">
              <Select value={categoryFilter} onValueChange={handleCategoryChange}>
                <SelectTrigger className="h-9 w-[140px] rounded-lg text-sm">
                  <SelectValue placeholder="All categories" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All</SelectItem>
                  {CATEGORIES.map((cat) => (
                    <SelectItem key={cat.id} value={String(cat.id)}>
                      {cat.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Button onClick={openCreate} size="default" className="shrink-0 rounded-lg">
                <Plus className="mr-2 h-4 w-4" />
                New Contact
              </Button>
            </div>
          )}
        </div>

        <TabsContent value="active" className="space-y-4">
          {/* Search bar */}
          <div className="max-w-sm">
            <SearchBar onSearch={handleSearch} />
          </div>

          {/* Search feedback */}
          {isSearching && !searchResults.isLoading && (
            <p className="text-xs text-muted-foreground">
              {activeContacts.length} {activeContacts.length === 1 ? "result" : "results"} for &ldquo;{searchQuery}&rdquo;
            </p>
          )}

          {/* Error */}
          {(isSearching ? searchResults.isError : contactsQuery.isError) ? (
            <div className="flex items-center justify-center py-12 text-destructive">
              Failed to load contacts.
            </div>
          ) : /* Empty states */
          !contactsQuery.isLoading && !isSearching && !activeContacts.length ? (
            <ContactsEmptyState type="no-contacts" onAddContact={openCreate} />
          ) : isSearching && !searchResults.isLoading && !activeContacts.length ? (
            <ContactsEmptyState
              type="no-results"
              searchTerm={searchQuery}
              onClearSearch={handleClearSearch}
            />
          ) : (
            <ContactsTable
              contacts={activeContacts}
              isLoading={isSearching ? searchResults.isLoading : contactsQuery.isLoading}
              onEdit={openEdit}
              onDelete={handleDelete}
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
          <Alert className="border-warning-muted bg-warning-muted text-foreground">
            <InfoIcon className="h-4 w-4 text-warning" />
            <AlertDescription className="ml-1 text-sm">
              Archived contacts will be permanently removed after 30 days.
            </AlertDescription>
          </Alert>

          {deletedQuery.isError ? (
            <div className="flex items-center justify-center py-12 text-destructive">
              Failed to load archived contacts.
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
