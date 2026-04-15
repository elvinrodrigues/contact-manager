import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { CATEGORIES } from "@/constants/categories";

const CATEGORY_STYLES: Record<string, string> = {
  Friends:
    "border-transparent bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300",
  Work: "border-transparent bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-300",
  Family:
    "border-transparent bg-yellow-100 text-yellow-800 dark:bg-yellow-900/40 dark:text-yellow-300",
  Default:
    "border-transparent bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400",
};

interface CategoryBadgeProps {
  categoryId?: number;
}

export function CategoryBadge({ categoryId }: CategoryBadgeProps) {
  const cat = CATEGORIES.find((c) => c.id === categoryId);
  const name = cat?.name ?? "Default";
  const styles = CATEGORY_STYLES[name] ?? CATEGORY_STYLES.Default;

  return (
    <Badge className={cn("font-medium", styles)} variant="outline">
      {name}
    </Badge>
  );
}
