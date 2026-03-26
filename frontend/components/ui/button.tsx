import * as React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "@/lib/utils";

const buttonVariants = cva(
  // ── Base ──────────────────────────────────────────────────────────────────
  [
    "inline-flex items-center justify-center gap-2 whitespace-nowrap",
    "rounded-xl", // rounder than the default rounded-md
    "text-sm font-semibold",
    "transition-all duration-150",
    "select-none",
    "outline-none",
    "focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1",
    "disabled:pointer-events-none disabled:opacity-50",
    "active:scale-[0.97]",
    // SVG children
    "[&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4 [&_svg]:shrink-0",
  ],
  {
    variants: {
      variant: {
        // ── Primary ──────────────────────────────────────────────────────────
        default: [
          "bg-primary text-primary-foreground",
          "shadow-sm shadow-primary/30",
          "hover:bg-primary/90 hover:shadow-md hover:shadow-primary/25",
          "active:shadow-none",
        ],

        // ── Destructive ──────────────────────────────────────────────────────
        destructive: [
          "bg-destructive text-white",
          "shadow-sm shadow-destructive/25",
          "hover:bg-destructive/90 hover:shadow-md hover:shadow-destructive/20",
          "focus-visible:ring-destructive/40",
          "dark:bg-destructive/80",
          "active:shadow-none",
        ],

        // ── Outline ──────────────────────────────────────────────────────────
        outline: [
          "border border-border bg-background text-foreground",
          "shadow-xs",
          "hover:bg-accent hover:text-accent-foreground hover:border-border",
          "dark:bg-input/20 dark:border-input dark:hover:bg-input/40",
        ],

        // ── Secondary ────────────────────────────────────────────────────────
        secondary: [
          "bg-secondary text-secondary-foreground",
          "shadow-xs",
          "hover:bg-secondary/70",
        ],

        // ── Ghost ────────────────────────────────────────────────────────────
        ghost: [
          "text-foreground",
          "hover:bg-accent hover:text-accent-foreground",
          "dark:hover:bg-accent/60",
        ],

        // ── Link ─────────────────────────────────────────────────────────────
        link: [
          "text-primary underline-offset-4",
          "hover:underline",
          "active:scale-100", // no squish for link style
        ],
      },

      size: {
        default: "h-9 px-4 py-2 has-[>svg]:px-3",
        sm: "h-8 px-3 text-xs gap-1.5 has-[>svg]:px-2.5",
        lg: "h-10 px-6 text-sm has-[>svg]:px-4",
        xl: "h-11 px-7 text-base has-[>svg]:px-5",
        icon: "size-9 rounded-xl",
        "icon-sm": "size-8 rounded-lg",
        "icon-lg": "size-10 rounded-xl",
      },
    },

    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);

function Button({
  className,
  variant,
  size,
  asChild = false,
  ...props
}: React.ComponentProps<"button"> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean;
  }) {
  const Comp = asChild ? Slot : "button";

  return (
    <Comp
      data-slot="button"
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  );
}

export { Button, buttonVariants };
