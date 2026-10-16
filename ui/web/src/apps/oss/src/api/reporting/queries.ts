import { PaginationParams } from "@plakar-ui/common/api/pagination";
import { queryOptions } from "@tanstack/react-query";

import { getReportingReportsList } from "./api";
import { reportingKeys } from "./keys";

export const listReportingReportsQueryOptions = ({
  repository,
  page,
  perPage,
}: { repository: string } & PaginationParams) =>
  queryOptions({
    queryKey: reportingKeys.list({ repository, page, perPage }),
    refetchInterval: 60 * 1000,
    queryFn: () =>
      getReportingReportsList({
        repository,
        pagination: { page, perPage },
      }),
  });
