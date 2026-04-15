import React from "react";
import { Link } from "react-router-dom";
import { UserPlus } from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import { Skeleton } from "@/components/ui/skeleton";
import type { Contact } from "@/types/contact";

interface ActivityFeedProps {
  contacts: Contact[];
  loading?: boolean;
}

const ActivityItem = React.memo(function ActivityItem({
  contact,
}: {
  contact: Contact;
}) {
  const timeAgo = contact.createdAt
    ? formatDistanceToNow(new Date(contact.createdAt), { addSuffix: true })
    : "";

  return (
    <div className="flex items-center gap-3 border-b border-border px-1 py-3 last:border-0">
      <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-success-muted">
        <UserPlus className="h-3.5 w-3.5 text-success" />
      </div>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm">
          <span className="font-medium">{contact.name}</span>{" "}
          <span className="text-muted-foreground">was added</span>
        </p>
      </div>
      {timeAgo && (
        <span className="shrink-0 text-xs text-muted-foreground">
          {timeAgo}
        </span>
      )}
    </div>
  );
});

export function ActivityFeed({ contacts, loading }: ActivityFeedProps) {
  if (loading) {
    return (
      <div className="rounded-xl border border-border bg-card p-5 shadow-card">
        <Skeleton className="mb-4 h-4 w-28" />
        <div className="space-y-1">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-10 w-full" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-border bg-card p-5 shadow-card">
      <p className="mb-3 text-sm font-semibold">Recent activity</p>

      {!contacts.length ? (
        <div className="py-8 text-center">
          <p className="text-sm text-muted-foreground">No recent activity</p>
          <p className="mt-1 text-xs text-muted-foreground">
            Contacts you create will appear here.
          </p>
        </div>
      ) : (
        <>
          <div>
            {contacts.slice(0, 8).map((c) => (
              <ActivityItem key={c.id} contact={c} />
            ))}
          </div>
          <Link
            to="/contacts"
            className="mt-3 block text-center text-xs text-muted-foreground transition-colors hover:text-foreground"
          >
            View all contacts →
          </Link>
        </>
      )}
    </div>
  );
}
