import { useMemo } from "react";
import { Users, CalendarPlus, FolderOpen, Archive } from "lucide-react";
import { useStats } from "@/hooks/use-contacts";
import { StatCard } from "@/components/dashboard/StatCard";
import { CategoryBreakdown } from "@/components/dashboard/CategoryBreakdown";
import { ActivityFeed } from "@/components/dashboard/ActivityFeed";

export default function Dashboard() {
  const { data: stats, isLoading } = useStats();

  const topCategory = useMemo(() => {
    if (!stats?.categories?.length) return null;
    const sorted = [...stats.categories].sort((a, b) => b.count - a.count);
    return sorted[0];
  }, [stats?.categories]);

  const archivedPercent =
    stats && stats.total + stats.deleted > 0
      ? ((stats.deleted / (stats.total + stats.deleted)) * 100).toFixed(0)
      : "0";

  const topCatPercent =
    topCategory && stats && stats.total > 0
      ? ((topCategory.count / stats.total) * 100).toFixed(0)
      : "0";

  return (
    <div className="space-y-6 animate-fade-in">
      {/* Stat cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          label="Total Contacts"
          value={stats?.total ?? 0}
          subtext={`${stats?.total ?? 0} active in your address book`}
          icon={Users}
          loading={isLoading}
        />
        <StatCard
          label="Added This Week"
          value={stats?.added_this_week ?? 0}
          subtext="New contacts this week"
          icon={CalendarPlus}
          loading={isLoading}
          trend={
            stats && stats.added_this_week > 0
              ? "up"
              : "neutral"
          }
          trendValue={
            stats && stats.added_this_week > 0
              ? `+${stats.added_this_week}`
              : undefined
          }
        />
        <StatCard
          label="Top Category"
          value={topCategory?.name ?? "—"}
          subtext={
            topCategory
              ? `${topCategory.count} contacts · ${topCatPercent}% of total`
              : "No categories yet"
          }
          icon={FolderOpen}
          loading={isLoading}
        />
        <StatCard
          label="Archived"
          value={stats?.deleted ?? 0}
          subtext={`${archivedPercent}% of total contacts`}
          icon={Archive}
          loading={isLoading}
        />
      </div>

      {/* Two-column: Categories + Activity */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-5">
        <div className="lg:col-span-3">
          <CategoryBreakdown
            categories={stats?.categories ?? []}
            total={stats?.total ?? 0}
            loading={isLoading}
          />
        </div>
        <div className="lg:col-span-2">
          <ActivityFeed
            contacts={stats?.recent ?? []}
            loading={isLoading}
          />
        </div>
      </div>
    </div>
  );
}
