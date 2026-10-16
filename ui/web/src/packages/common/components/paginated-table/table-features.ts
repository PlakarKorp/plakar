import {
  columnVisibilityFeature,
  createExpandedRowModel,
  createPaginatedRowModel,
  rowExpandingFeature,
  rowPaginationFeature,
  rowSelectionFeature,
  tableFeatures,
} from "@tanstack/react-table";

/**
 * Every table in the app is built from the same feature set. Only row
 * selection, row expanding, row pagination, and column visibility (via
 * row.getVisibleCells() in the shared row renderer) are actually used
 * anywhere -- no column filtering, sorting, grouping, pinning, resizing,
 * sizing, faceting, global filtering, row pinning, row aggregation, or
 * cell selection/spanning is used, so those stock features are left out to
 * keep this tree-shakeable. The expanded row model is included so tables
 * backed by hierarchical data (via `getSubRows`) can flatten/collapse
 * their subrows.
 *
 * The paginated row model serves the tables whose rows all arrive at once. It
 * changes nothing for the others: a table setting `manualPagination` never
 * reaches it.
 */
export const appTableFeatures = tableFeatures({
  rowSelectionFeature,
  rowExpandingFeature,
  rowPaginationFeature,
  columnVisibilityFeature,
  expandedRowModel: createExpandedRowModel(),
  paginatedRowModel: createPaginatedRowModel(),
});

export type AppTableFeatures = typeof appTableFeatures;
