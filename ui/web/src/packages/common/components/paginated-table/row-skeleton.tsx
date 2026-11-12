import { Checkbox, Skeleton } from "@plakar-ui/ui/components";
import { ReactTable as TanstackTable, RowData } from "@tanstack/react-table";
import clsx from "clsx";
import { Cell, Row as TableRow } from "react-aria-components";

import { AppTableFeatures } from "./table-features";

type TableRowsSkeletonProps<T extends RowData> = {
  table: TanstackTable<AppTableFeatures, T>;
  hasExpandColumn?: boolean;
};

export function TableRowsSkeleton<T extends RowData>({
  table,
  hasExpandColumn,
}: TableRowsSkeletonProps<T>) {
  // Use the number of rows in the current page, or fallback to the page size
  const length =
    table.getRowModel().rows.length || table.state.pagination.pageSize;

  return [...Array(length)].map((_, index) => (
    <TableRow
      key={index}
      className="border-secondary hover:bg-primary-hover group cursor-pointer border-b last:border-b-0"
    >
      {table.options.enableMultiRowSelection && (
        <Cell className="w-4 p-2">
          <Checkbox isDisabled />
        </Cell>
      )}
      {hasExpandColumn && <Cell className="w-6 p-2 pl-4" />}
      {table.getVisibleLeafColumns().map((cell) => (
        <Cell
          key={cell.id}
          className={clsx("p-2", cell.columnDef.meta?.cellClassName)}
        >
          {cell.columnDef.meta?.skeleton || (
            <Skeleton variant="text" size="md" />
          )}
        </Cell>
      ))}
    </TableRow>
  ));
}
