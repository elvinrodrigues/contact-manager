import { Search, BookOpen } from "lucide-react";
import { Button } from "@/components/ui/button";

interface ContactsEmptyStateProps {
  type: "no-contacts" | "no-results";
  searchTerm?: string;
  onAddContact?: () => void;
  onClearSearch?: () => void;
}

export function ContactsEmptyState({
  type,
  searchTerm,
  onAddContact,
  onClearSearch,
}: ContactsEmptyStateProps) {
  return (
    <div className="flex flex-col items-center gap-3 py-16">
      <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-surface-muted">
        {type === "no-results" ? (
          <Search className="h-7 w-7 text-muted-foreground" />
        ) : (
          <BookOpen className="h-7 w-7 text-muted-foreground" />
        )}
      </div>

      {type === "no-results" ? (
        <>
          <p className="text-sm font-medium">
            No results for &ldquo;{searchTerm}&rdquo;
          </p>
          <p className="text-sm text-muted-foreground">
            Try a different name, email, or category.
          </p>
          {onClearSearch && (
            <Button variant="ghost" size="sm" onClick={onClearSearch}>
              Clear search
            </Button>
          )}
        </>
      ) : (
        <>
          <p className="text-sm font-medium">No contacts yet</p>
          <p className="text-sm text-muted-foreground">
            Add your first contact to get started.
          </p>
          {onAddContact && (
            <Button size="sm" onClick={onAddContact}>
              + Add Contact
            </Button>
          )}
        </>
      )}
    </div>
  );
}
