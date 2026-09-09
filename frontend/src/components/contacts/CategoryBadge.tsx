import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { categoryName } from "@/constants/categories";

// Keyed by the category names seeded in backend/migrations/000_init.sql.
const CATEGORY_STYLES: Record<string, string> = {
  Friends:
    "border-transparent bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300",
  Work: "border-transparent bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-300",
  Family:
    "border-transparent bg-yellow-100 text-yellow-800 dark:bg-yellow-900/40 dark:text-yellow-300",
  College:
    "border-transparent bg-purple-100 text-purple-800 dark:bg-purple-900/40 dark:text-purple-300",
  General:
    "border-transparent bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400",
};

const FALLBACK_STYLE = CATEGORY_STYLES.General;

interface CategoryBadgeProps {
  categoryId?: number;
}

export function CategoryBadge({ categoryId }: CategoryBadgeProps) {
  // An unknown id still renders a readable label rather than a wrong one.
  const name = categoryId === undefined ? "General" : categoryName(categoryId);
  const styles = CATEGORY_STYLES[name] ?? FALLBACK_STYLE;

  return (
    <Badge className={cn("font-medium", styles)} variant="outline">
      {name}
    </Badge>
  );
}
