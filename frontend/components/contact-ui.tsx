import { cn } from "@/lib/utils";

// ─── Category map ──────────────────────────────────────────────────────────

export const CATEGORY_MAP: Record<
  number,
  { name: string; badgeClass: string }
> = {
  1: {
    name: "Default",
    badgeClass:
      "bg-slate-100 text-slate-600 border-slate-200 dark:bg-slate-800/60 dark:text-slate-300 dark:border-slate-700",
  },
  2: {
    name: "Family",
    badgeClass:
      "bg-rose-50 text-rose-700 border-rose-200 dark:bg-rose-900/30 dark:text-rose-400 dark:border-rose-800/60",
  },
  3: {
    name: "Friends",
    badgeClass:
      "bg-sky-50 text-sky-700 border-sky-200 dark:bg-sky-900/30 dark:text-sky-400 dark:border-sky-800/60",
  },
  4: {
    name: "Work",
    badgeClass:
      "bg-violet-50 text-violet-700 border-violet-200 dark:bg-violet-900/30 dark:text-violet-400 dark:border-violet-800/60",
  },
};

// ─── CategoryBadge ─────────────────────────────────────────────────────────

interface CategoryBadgeProps {
  categoryId?: number;
  className?: string;
}

export function CategoryBadge({ categoryId, className }: CategoryBadgeProps) {
  const cat = categoryId != null ? CATEGORY_MAP[categoryId] : null;

  if (!cat) {
    return (
      <span className="text-muted-foreground/60 text-xs italic">—</span>
    );
  }

  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium tracking-wide whitespace-nowrap",
        cat.badgeClass,
        className
      )}
    >
      {cat.name}
    </span>
  );
}

// ─── Avatar colour palette ────────────────────────────────────────────────

const AVATAR_COLORS = [
  "bg-indigo-500",
  "bg-violet-500",
  "bg-rose-500",
  "bg-sky-500",
  "bg-emerald-500",
  "bg-amber-500",
  "bg-orange-500",
  "bg-pink-500",
  "bg-teal-500",
  "bg-cyan-500",
];

// ─── ContactAvatar ────────────────────────────────────────────────────────

interface ContactAvatarProps {
  name: string;
  /** Size in pixels — defaults to 32 (w-8 h-8) */
  size?: "sm" | "md" | "lg";
  className?: string;
}

const sizeClasses = {
  sm: "w-7 h-7 text-[10px]",
  md: "w-8 h-8 text-xs",
  lg: "w-10 h-10 text-sm",
};

export function ContactAvatar({
  name,
  size = "md",
  className,
}: ContactAvatarProps) {
  // Build initials — max 2 chars
  const initials = name
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase() ?? "")
    .join("");

  // Deterministic colour from the first character's code point
  const colorIndex =
    (name.codePointAt(0) ?? 0) % AVATAR_COLORS.length;
  const colorClass = AVATAR_COLORS[colorIndex];

  return (
    <div
      aria-hidden="true"
      className={cn(
        "rounded-full flex items-center justify-center flex-shrink-0",
        "text-white font-semibold select-none leading-none",
        "ring-2 ring-background", // small white ring so avatar lifts off the row
        sizeClasses[size],
        colorClass,
        className
      )}
    >
      {initials || "?"}
    </div>
  );
}
