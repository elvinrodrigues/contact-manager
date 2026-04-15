import { cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";
import type { LucideIcon } from "lucide-react";

interface StatCardProps {
  label: string;
  value: number | string;
  subtext: string;
  trend?: "up" | "down" | "neutral";
  trendValue?: string;
  icon: LucideIcon;
  loading?: boolean;
}

export function StatCard({
  label,
  value,
  subtext,
  trend,
  trendValue,
  icon: Icon,
  loading,
}: StatCardProps) {
  if (loading) {
    return (
      <div className="rounded-xl border border-border bg-card p-5 shadow-card">
        <div className="flex items-start gap-4">
          <Skeleton className="h-10 w-10 rounded-lg" />
          <div className="flex-1 space-y-2">
            <Skeleton className="h-3.5 w-24" />
            <Skeleton className="h-8 w-16" />
            <Skeleton className="h-3 w-32" />
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-border bg-card p-5 shadow-card transition-all duration-200 hover:-translate-y-0.5 hover:shadow-card-hover">
      <div className="flex items-start gap-4">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-accent-brand-muted">
          <Icon className="h-5 w-5 text-primary" />
        </div>
        <div className="min-w-0 flex-1">
          <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
            {label}
          </p>
          <div className="mt-1 flex items-baseline gap-2">
            <p className="text-3xl font-semibold tabular-nums">{value}</p>
            {trend && trendValue && (
              <span
                className={cn(
                  "inline-flex rounded-full px-2 py-0.5 text-xs font-medium",
                  trend === "up" && "bg-success-muted text-success",
                  trend === "down" && "bg-danger-muted text-danger",
                  trend === "neutral" && "bg-muted text-muted-foreground"
                )}
              >
                {trendValue}
              </span>
            )}
          </div>
          <p className="mt-1 text-xs text-muted-foreground">{subtext}</p>
        </div>
      </div>
    </div>
  );
}
