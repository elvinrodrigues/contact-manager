// Mirrors the seed in backend/migrations/000_init.sql. Ids are stable because
// the seed uses ON CONFLICT (name) DO NOTHING, so a category keeps its id once
// created. Category 1 is "General" in the database — the UI previously labelled
// it "Default" and omitted "College" entirely, so contacts in category 5
// rendered with no label at all.
export const CATEGORIES = [
  { id: 1, name: "General" },
  { id: 2, name: "Family" },
  { id: 3, name: "Friends" },
  { id: 4, name: "Work" },
  { id: 5, name: "College" },
] as const;

export const DEFAULT_CATEGORY_ID = 1;

/** Falls back to the id itself so an unknown category is still identifiable. */
export function categoryName(id: number): string {
  return CATEGORIES.find((c) => c.id === id)?.name ?? `Category ${id}`;
}
