import { cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";

interface CategoryItem {
  name: string;
  count: number;
}

interface CategoryBreakdownProps {
  categories: CategoryItem[];
  total: number;
  loading?: boolean;
}

const CATEGORY_COLORS = [
  "bg-blue-500",
  "bg-emerald-500",
  "bg-amber-500",
  "bg-violet-500",
  "bg-rose-500",
  "bg-cyan-500",
  "bg-orange-500",
];

function hashColor(name: string): string {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }
  return CATEGORY_COLORS[Math.abs(hash) % CATEGORY_COLORS.length];
}

export function CategoryBreakdown({
  categories,
  total,
  loading,
}: CategoryBreakdownProps) {
  if (loading) {
    return (
      <div className="rounded-xl border border-border bg-card p-5 shadow-card">
        <div className="mb-4 flex items-center justify-between">
          <Skeleton className="h-4 w-36" />
          <Skeleton className="h-3.5 w-20" />
        </div>
        <div className="space-y-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-6 w-full" />
          ))}
        </div>
      </div>
    );
  }

  if (!categories.length) {
    return (
      <div className="rounded-xl border border-border bg-card p-5 shadow-card">
        <p className="text-sm font-semibold">Category breakdown</p>
        <p className="mt-6 text-center text-sm text-muted-foreground">
          No categories yet — add contacts to see distribution
        </p>
      </div>
    );
  }

  const sorted = [...categories].sort((a, b) => b.count - a.count);
  const largestName = sorted[0]?.name;
  const visible = sorted.slice(0, 6);
  const hiddenCount = sorted.length - 6;

  return (
    <div className="rounded-xl border border-border bg-card p-5 shadow-card">
      <div className="mb-4 flex items-center justify-between">
        <p className="text-sm font-semibold">Category breakdown</p>
        <span className="text-xs text-muted-foreground">
          {categories.length} {categories.length === 1 ? "category" : "categories"} total
        </span>
      </div>
      <div className="space-y-3">
        {visible.map((cat) => {
          const pct = total > 0 ? (cat.count / total) * 100 : 0;
          const color = hashColor(cat.name);
          const isLargest = cat.name === largestName;

          return (
            <div
              key={cat.name}
              className={cn(
                "flex items-center gap-3 rounded-lg px-2 py-1.5 -mx-2",
                isLargest && "bg-accent-brand-muted/30"
              )}
            >
              <div className={cn("h-2.5 w-2.5 shrink-0 rounded-full", color)} />
              <span
                className={cn(
                  "flex-1 text-sm",
                  isLargest ? "font-semibold text-primary" : "font-medium"
                )}
              >
                {cat.name}
              </span>
              <span className="w-8 text-right text-sm tabular-nums text-muted-foreground">
                {cat.count}
              </span>
              <div className="mx-2 h-1.5 flex-1 overflow-hidden rounded-full bg-surface-muted">
                <div
                  className={cn(
                    "h-full rounded-full transition-all duration-500",
                    color
                  )}
                  style={{ width: `${Math.max(pct, 2)}%` }}
                />
              </div>
              <span className="w-10 text-right text-xs tabular-nums text-muted-foreground">
                {pct.toFixed(0)}%
              </span>
            </div>
          );
        })}
        {hiddenCount > 0 && (
          <p className="px-2 text-xs text-muted-foreground">
            +{hiddenCount} more {hiddenCount === 1 ? "category" : "categories"}
          </p>
        )}
      </div>
    </div>
  );
}
