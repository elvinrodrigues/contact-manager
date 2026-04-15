import { useState, useEffect } from "react";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogCancel,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import type { Contact, ContactFormData } from "@/types/contact";

export interface DuplicateData {
  existing: Contact;
  incoming: Contact;
  status: "duplicate" | "deleted_duplicate";
}

interface DuplicateDialogProps {
  data: DuplicateData | null;
  onClose: () => void;
  onOverwrite: (merged: ContactFormData) => void;
  onRestore: (merged: ContactFormData) => void;
}

export function DuplicateDialog({ data, onClose, onOverwrite, onRestore }: DuplicateDialogProps) {
  const [selection, setSelection] = useState({
    name: "existing",
    phone: "existing",
    email: "existing",
    category: "existing",
  });

  // Reset selection when data changes
  useEffect(() => {
    if (data) {
      setSelection({
        name: "existing",
        phone: "existing",
        email: "existing",
        category: "existing",
      });
    }
  }, [data]);

  const getCategoryName = (id?: number) => {
    switch (id) {
      case 1: return "Default";
      case 2: return "Family";
      case 3: return "Friends";
      case 4: return "Work";
      default: return "Default";
    }
  };

  if (!data) return null;

  const { existing, incoming, status } = data;

  const handleSave = (action: "overwrite" | "restore") => {
    const merged: ContactFormData = {
      name: selection.name === "existing" ? existing.name : incoming.name,
      phone: selection.phone === "existing" ? existing.phone : incoming.phone,
      email: selection.email === "existing" ? existing.email : incoming.email,
      categoryId: selection.category === "existing" ? existing.categoryId : incoming.categoryId,
    };
    
    if (action === "overwrite") {
      onOverwrite(merged);
    } else {
      onRestore(merged);
    }
  };

  const renderField = (label: string, field: "name" | "phone" | "email" | "category", existingVal: any, incomingVal: any) => {
    const isSame = existingVal === incomingVal;
    const disableField = field !== "category" && isSame;
    
    return (
      <div className="grid grid-cols-[100px_1fr_1fr] items-center gap-4 py-3 border-b last:border-0 border-border/50">
        <span className="font-medium text-sm text-muted-foreground">{label}</span>
        
        <label className={`flex items-center gap-2 text-sm cursor-pointer ${disableField ? 'opacity-70 cursor-not-allowed' : ''}`}>
          <input
            type="radio"
            name={`${field}-existing`}
            value="existing"
            checked={selection[field as keyof typeof selection] === "existing"}
            onChange={() => setSelection({ ...selection, [field]: "existing" })}
            disabled={disableField}
            className="accent-primary"
          />
          <span className={selection[field as keyof typeof selection] === "existing" && !disableField ? "font-bold text-foreground" : ""}>
            {existingVal || "—"}
          </span>
        </label>

        <label className={`flex items-center gap-2 text-sm cursor-pointer ${disableField ? 'opacity-70 cursor-not-allowed' : ''}`}>
          <input
            type="radio"
            name={`${field}-incoming`}
            value="incoming"
            checked={selection[field as keyof typeof selection] === "incoming"}
            onChange={() => setSelection({ ...selection, [field]: "incoming" })}
            disabled={disableField}
            className="accent-primary"
          />
          <span className={selection[field as keyof typeof selection] === "incoming" && !disableField ? "font-bold text-primary" : ""}>
            {incomingVal || "—"}
          </span>
        </label>
      </div>
    );
  };

  return (
    <AlertDialog open={!!data} onOpenChange={(o) => !o && onClose()}>
      <AlertDialogContent className="max-w-2xl">
        <AlertDialogHeader>
          <AlertDialogTitle>Duplicate Contact Found</AlertDialogTitle>
          <AlertDialogDescription>
            {status === "deleted_duplicate" 
              ? "This contact was previously deleted. Map any updated fields before restoring."
              : "A contact with this phone number already exists. Select which fields to keep, then save."}
          </AlertDialogDescription>
        </AlertDialogHeader>
        
        <div className="py-2">
          <div className="grid grid-cols-[100px_1fr_1fr] gap-4 px-4 mb-2">
            <span className="font-semibold text-xs text-muted-foreground uppercase">Field</span>
            <span className="font-semibold text-xs text-muted-foreground uppercase">Existing Contact</span>
            <span className="font-semibold text-xs border-l border-primary/20 pl-4 text-primary uppercase">Incoming Data</span>
          </div>
          
          <div className="rounded-md border p-4 bg-muted/10">
            {renderField("Name", "name", existing.name, incoming.name)}
            {renderField("Phone", "phone", existing.phone, incoming.phone)}
            {renderField("Email", "email", existing.email, incoming.email)}
            {renderField("Category", "category", getCategoryName(existing.categoryId), getCategoryName(incoming.categoryId))}
          </div>
        </div>

        <AlertDialogFooter className="sm:justify-between items-center mt-2">
          <AlertDialogCancel onClick={onClose}>Cancel</AlertDialogCancel>
          <div className="flex gap-2">
            <Button variant="outline" onClick={onClose}>
              Keep Existing
            </Button>
            {status === "deleted_duplicate" ? (
              <Button onClick={() => handleSave("restore")} className="bg-green-600 hover:bg-green-700">
                Restore & Merge
              </Button>
            ) : (
              <Button onClick={() => handleSave("overwrite")} variant="default">
                Save Merged
              </Button>
            )}
          </div>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
