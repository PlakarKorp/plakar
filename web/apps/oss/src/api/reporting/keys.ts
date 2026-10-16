import type { PaginationParams } from "@plakar-ui/common/api/pagination";

type ListReportingReportsParams = { repository: string } & PaginationParams;

export const reportingKeys = {
  all: () => ["api", "api.plakar.io", "reporting"] as const,
  lists: () => [...reportingKeys.all(), "list"] as const,
  list: (params: ListReportingReportsParams) =>
    [...reportingKeys.lists(), params] as const,
};
