"use client";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Spinner } from "@/components/ui/spinner";
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { AlertTriangle, Info } from "lucide-react";

interface ConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: string;
  confirmLabel?: string;
  cancelLabel?: string;
  onConfirm: () => void;
  isLoading?: boolean;
  variant?: "destructive" | "default";
}

export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel = "Confirm",
  cancelLabel = "Cancel",
  onConfirm,
  isLoading = false,
  variant = "destructive",
}: ConfirmDialogProps) {
  const isDestructive = variant === "destructive";

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent className="p-0 overflow-hidden gap-0 max-w-md">
        {/* Header strip with icon */}
        <div className="px-6 pt-6 pb-5">
          <div className="flex items-start gap-4">
            {/* Icon bubble */}
            <div
              className={cn(
                "flex-shrink-0 w-10 h-10 rounded-full flex items-center justify-center",
                isDestructive
                  ? "bg-destructive/10 text-destructive"
                  : "bg-primary/10 text-primary"
              )}
            >
              {isDestructive ? (
                <AlertTriangle size={18} strokeWidth={2.2} />
              ) : (
                <Info size={18} strokeWidth={2.2} />
              )}
            </div>

            {/* Title + description */}
            <AlertDialogHeader className="gap-1 text-left p-0">
              <AlertDialogTitle className="text-base font-semibold leading-snug">
                {title}
              </AlertDialogTitle>
              <AlertDialogDescription className="text-sm leading-relaxed">
                {description}
              </AlertDialogDescription>
            </AlertDialogHeader>
          </div>
        </div>

        {/* Divider */}
        <div className="h-px bg-border mx-0" />

        {/* Footer */}
        <AlertDialogFooter className="px-6 py-4 bg-muted/30 flex flex-row justify-end gap-2">
          <AlertDialogCancel
            disabled={isLoading}
            className={cn(
              buttonVariants({ variant: "outline" }),
              "h-9 px-4 text-sm font-medium transition-colors"
            )}
          >
            {cancelLabel}
          </AlertDialogCancel>

          <AlertDialogAction
            onClick={(e) => {
              e.preventDefault();
              onConfirm();
            }}
            disabled={isLoading}
            className={cn(
              isDestructive
                ? buttonVariants({ variant: "destructive" })
                : buttonVariants({ variant: "default" }),
              "h-9 px-4 text-sm font-medium min-w-[90px] transition-all"
            )}
          >
            {isLoading ? (
              <span className="flex items-center gap-2">
                <Spinner className="h-3.5 w-3.5" />
                <span>Please wait…</span>
              </span>
            ) : (
              confirmLabel
            )}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
