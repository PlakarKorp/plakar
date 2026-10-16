import { Icon, Typography } from "@plakar-ui/ui/components";
import { UseQueryResult } from "@tanstack/react-query";
import {
  createColumnHelper,
  useTable as useReactTable,
} from "@tanstack/react-table";

import {
  getSnapshotPathErrorsFn,
  SnapshotPathError,
} from "../../../api/kloset";
import { PaginatedTable } from "../../paginated-table/paginated-table";
import {
  AppTableFeatures,
  appTableFeatures,
} from "../../paginated-table/table-features";

type ErrorsBrowserProps = {
  query: UseQueryResult<Awaited<ReturnType<getSnapshotPathErrorsFn>>>;
  page: number;
  perPage: number;
  onPageChange: (page: number, per_page: number) => void;
};

export function ErrorsBrowser({
  query,
  page,
  perPage,
  onPageChange,
}: ErrorsBrowserProps) {
  const columnHelper = createColumnHelper<
    AppTableFeatures,
    SnapshotPathError
  >();

  const columns = columnHelper.columns([
    columnHelper.accessor("name", {
      header: () => "Filename",
      meta: {
        cellClassName: "min-w-80 md:min-w-none md:w-1/3",
      },
      cell: ({ row }) => {
        return (
          <div className="flex gap-2">
            <Icon type="exclamation-circle" className="text-error" size="md" />
            <Typography
              weight="medium"
              size="md"
              className="text-primary leading-none break-all select-text"
            >
              {row.original.name}
            </Typography>
          </div>
        );
      },
    }),
    columnHelper.accessor("error", {
      header: () => "Error",
      meta: {
        cellClassName: "min-w-80 md:min-w-none",
      },
      cell: ({ row }) => {
        return (
          <Typography
            weight="normal"
            className="text-secondary break-all select-text"
          >
            {row.original.error}
          </Typography>
        );
      },
    }),
  ]);

  const paginationState = {
    pageIndex: page,
    pageSize: perPage,
  };

  const table = useReactTable({
    features: appTableFeatures,
    data: query.data?.items ?? [],
    rowCount: query.data?.total,
    columns,
    state: {
      pagination: paginationState,
    },
    getRowId: (_, idx) => {
      // We can't use row.name as an ID because it's not unique.
      return `row-${idx}`;
    },
    enableMultiRowSelection: false, // Only allow single row selection
    onPaginationChange: (pagination) => {
      const state =
        typeof pagination === "function"
          ? pagination(paginationState)
          : pagination;

      onPageChange(state.pageIndex, state.pageSize);
    },
    manualPagination: true,
  });

  return (
    <PaginatedTable<SnapshotPathError>
      variant="cards"
      query={query}
      table={table}
      ariaLabel="Restore point errors"
    />
  );
}
