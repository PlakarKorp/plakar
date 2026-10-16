import { Pagination as UIPagination } from "@plakar-ui/ui/components";
import { UseQueryResult } from "@tanstack/react-query";
import { ReactTable as TanstackTable, RowData } from "@tanstack/react-table";
import { useCallback } from "react";

import { AppTableFeatures } from "./table-features";
import {
  isPaginationResponseWithTotal,
  PaginationResponse,
} from "../../api/pagination";

interface PaginationProps<T extends RowData> {
  query: UseQueryResult<PaginationResponse<T>>;
  table: TanstackTable<AppTableFeatures, T>;
}

export function Pagination<T extends RowData>({
  query,
  table,
}: PaginationProps<T>) {
  const { data } = query;
  const { pageIndex, pageSize } = table.state.pagination;

  const hasPrev = pageIndex > 0;
  const hasNext = Boolean(
    data &&
    (isPaginationResponseWithTotal(data)
      ? (pageIndex + 1) * pageSize < data.total
      : data.has_next)
  );

  const navigateTo = useCallback(
    (newPage: number) => {
      table.setPageIndex(newPage);
    },
    [table]
  );
  const handlePrev = hasPrev ? () => navigateTo(pageIndex - 1) : undefined;
  const handleNext = hasNext ? () => navigateTo(pageIndex + 1) : undefined;
  const handlePageSizeChange = (perPage: number) => table.setPageSize(perPage);

  const total = {
    count: data && isPaginationResponseWithTotal(data) ? data.total : undefined,
    perPage: pageSize,
  };

  return (
    <UIPagination
      currentPage={pageIndex}
      onPrev={handlePrev}
      onNext={handleNext}
      onPageSizeChange={handlePageSizeChange}
      total={total}
    />
  );
}
