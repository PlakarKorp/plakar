import { Checkbox, Skeleton } from "@plakar-ui/ui/components";
import { ReactTable as TanstackTable, RowData } from "@tanstack/react-table";
import clsx from "clsx";
import { Cell, Row as TableRow } from "react-aria-components";

import { PaginatedTableVariant } from "./paginated-table";
import { AppTableFeatures } from "./table-features";

type TableRowsSkeletonProps<T extends RowData> = {
  variant: PaginatedTableVariant;
  table: TanstackTable<AppTableFeatures, T>;
  type: SkeletonType;
  hasExpandColumn?: boolean;
};

export type SkeletonType = "one-row" | "two-rows" | "three-rows";

export function TableRowsSkeleton<T extends RowData>({
  variant,
  table,
  type,
  hasExpandColumn,
}: TableRowsSkeletonProps<T>) {
  return {
    default: (
      <TableRowsSkeletonDefault
        table={table}
        type={type}
        hasExpandColumn={hasExpandColumn}
      />
    ),
    cards: <TableRowsSkeletonCards table={table} type={type} />,
  }[variant];
}

type TableRowsSkeletonVariantProps<T extends RowData> = Omit<
  TableRowsSkeletonProps<T>,
  "variant"
>;

function TableRowsSkeletonDefault<T extends RowData>({
  table,
  type,
  hasExpandColumn,
}: TableRowsSkeletonVariantProps<T>) {
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
          {cell.columnDef.meta?.skeleton ? (
            cell.columnDef.meta.skeleton
          ) : (
            <Row type={type} />
          )}
        </Cell>
      ))}
    </TableRow>
  ));
}

function TableRowsSkeletonCards<T extends RowData>({
  table,
  type,
}: TableRowsSkeletonVariantProps<T>) {
  const cellsCount = table.getVisibleLeafColumns().length;

  return [...Array(table.state.pagination.pageSize)].map((_, index) => (
    <TableRow key={index}>
      {table.getVisibleLeafColumns().map((cell, idx) => (
        <Cell
          key={cell.id}
          className={clsx(
            cell.columnDef.meta?.cellClassName,

            "border-secondary bg-primary border-t border-b p-3",
            idx === 0 && "rounded-l-lg border-l",
            idx === cellsCount - 1 && "rounded-r-lg border-r"
          )}
        >
          {cell.columnDef.meta?.skeleton ? (
            cell.columnDef.meta.skeleton
          ) : (
            <Row type={type} />
          )}
        </Cell>
      ))}
    </TableRow>
  ));
}

function Row({ type }: { type: SkeletonType }) {
  return (
    <>
      <Skeleton variant="text" size="md" />
      {type === "two-rows" && (
        <Skeleton variant="text" size="sm" className="mt-1" />
      )}
      {type === "three-rows" && (
        <>
          <Skeleton variant="text" size="xs" className="mt-1" />
          <Skeleton variant="text" size="sm" className="mt-2" />
        </>
      )}
    </>
  );
}
