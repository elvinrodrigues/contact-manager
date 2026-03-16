"use client";

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";

import { Button } from "@/components/ui/button";
import { Contact } from "@/lib/api-service";

interface DuplicateDialogProps {
  open: boolean;
  duplicates: Contact[];
  onCancel: () => void;
}

export function DuplicateDialog({
  open,
  duplicates,
  onCancel,
}: DuplicateDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onCancel}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Duplicate Contact Detected</DialogTitle>
        </DialogHeader>

        <div className="space-y-3">
          <p className="text-sm text-muted-foreground">
            Contact with the same phone number already exist:
          </p>

          {duplicates.map((contact) => (
            <div key={contact.id} className="border rounded-md p-3 text-sm">
              <div className="font-medium">{contact.name}</div>
              <div>{contact.phone}</div>
              {contact.email && <div>{contact.email}</div>}
            </div>
          ))}
        </div>

        <DialogFooter className="flex gap-2">
          <Button variant="outline" onClick={onCancel}>
            Cancel
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
