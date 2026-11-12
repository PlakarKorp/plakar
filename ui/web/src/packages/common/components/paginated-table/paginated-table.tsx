import {
  Checkbox,
  EmptyList,
  Icon,
  Typography,
} from "@plakar-ui/ui/components";
import { UseQueryResult } from "@tanstack/react-query";
import {
  flexRender,
  ReactTable as TanstackTable,
  Row,
  RowData,
  TableFeatures,
} from "@tanstack/react-table";
import clsx from "clsx";
import { Dispatch, Fragment, memo, MouseEvent, ReactNode } from "react";
import {
  Cell,
  Column,
  Row as TableRow,
  Table,
  TableBody,
  TableHeader,
  LinkProps,
} from "react-aria-components";

import { Pagination } from "./pagination";
import { TableRowsSkeleton } from "./row-skeleton";
import { AppTableFeatures } from "./table-features";
import { PaginationResponse } from "../../api/pagination";

import "@tanstack/react-table";

declare module "@tanstack/react-table" {
  /* eslint-disable @typescript-eslint/no-unused-vars, unused-imports/no-unused-vars */
  interface ColumnMeta<
    TFeatures extends TableFeatures,
    TData extends RowData,
    TValue,
  > {
    cellClassName?: string;
    skeleton?: React.ReactNode;
  }
  /* eslint-enable @typescript-eslint/no-unused-vars, unused-imports/no-unused-vars */
}

/**
 * Horizontal space, in rem, taken by a tree chevron and its gap.
 *
 * Also used as the indentation step between two nesting levels, so a child
 * chevron lines up with the start of its parent label.
 */
const TREE_INDENT_REM = 1.25;

type PaginatedTableProps<T extends RowData> = {
  query: UseQueryResult<PaginationResponse<T>>;
  table: TanstackTable<AppTableFeatures, T>;
  softSelectionState?: [T | null, Dispatch<T | null>];
  isSoftSelectionEnabledForRow?: (row: T) => boolean;
  filters?: React.ReactNode;
  ariaLabel?: string;
  getLink?: (row: T) => LinkProps["href"];
  isLoading?: boolean;
  disablePagination?: boolean;
  getExpandedContent?: (row: T) => ReactNode;
  getEmptyContent?: () => React.ReactNode;
  onRowContextMenu?: (row: T, event: MouseEvent) => void;

  /**
   * Group of a row, for flat tables. A header row is rendered before each row
   * whose group key differs from the previous row's, on the current page only.
   */
  getGroup?: (row: T) => { key: string; label: ReactNode } | undefined;
};

export function PaginatedTable<T extends RowData>({
  query,
  table,
  softSelectionState,
  isSoftSelectionEnabledForRow,
  filters,
  ariaLabel,
  getLink,
  isLoading = false,
  disablePagination = false,
  getExpandedContent,
  getEmptyContent,
  onRowContextMenu,
  getGroup,
}: PaginatedTableProps<T>) {
  const handleRetry = () => {
    query.refetch();
  };

  // Tree tables render their chevron inline in the first cell so that it is
  // indented along with the row content, instead of in a dedicated column whose
  // shared width would shift every row as soon as a nested one is expanded.
  const isTree = !!table.options.getSubRows;
  const hasExpandColumn = !!getExpandedContent && !isTree;

  const extraColCount =
    (table.options.enableMultiRowSelection ? 1 : 0) + (hasExpandColumn ? 1 : 0);
  const colCount = table.getVisibleLeafColumns().length + extraColCount;

  const rows = table.getRowModel().rows;
  const groups =
    getGroup && !isTree ? rows.map((row) => getGroup(row.original)) : [];

  return (
    <div className="flex min-h-[300px] flex-col gap-2">
      {filters && (
        <div className="flex items-end gap-2 has-[[data-filter-panel]]:pb-2">
          <div className="flex min-w-0 flex-1 flex-wrap items-center gap-2">
            {filters}
          </div>
        </div>
      )}

      <div
        className="border-secondary overflow-auto rounded-lg border"
        onKeyDownCapture={stopTypeaheadOnTextEntry}
      >
        <Table
          key={query.isPending || isLoading ? "loading" : "data"}
          aria-label={ariaLabel}
          className="bg-primary w-full table-auto border-collapse"
        >
          <TableHeader className="border-secondary bg-tertiary border-b text-left">
            {table.options.enableMultiRowSelection && (
              <Column className="w-4 p-2">
                <Checkbox
                  slot="selection"
                  isDisabled={table
                    .getRowModel()
                    .rows.every((row) => !row.getCanSelect())}
                  isSelected={table.getIsAllRowsSelected()}
                  onChange={table.toggleAllRowsSelected}
                />
              </Column>
            )}
            {hasExpandColumn && <Column className="w-6 p-2" />}
            {table.getHeaderGroups().map((headerGroup) =>
              headerGroup.headers.map((header, index) => (
                <Column
                  key={header.id}
                  isRowHeader={index === 0}
                  className={clsx(
                    "p-2",
                    header.column.columnDef.meta?.cellClassName
                  )}
                  style={
                    isTree && index === 0
                      ? { paddingLeft: `${0.5 + TREE_INDENT_REM}rem` }
                      : undefined
                  }
                >
                  <Typography
                    variant="body"
                    size="sm"
                    className="text-tertiary flex h-6 items-center"
                    weight="medium"
                  >
                    {flexRender(
                      header.column.columnDef.header,
                      header.getContext()
                    )}
                  </Typography>
                </Column>
              ))
            )}
          </TableHeader>

          <TableBody>
            {query.isPending || isLoading ? (
              <TableRowsSkeleton
                table={table}
                hasExpandColumn={hasExpandColumn}
              />
            ) : query.isError ? (
              <TableRow>
                <Cell colSpan={colCount} className="py-2">
                  <EmptyList
                    illustrationType="sad"
                    title="Failed to load data"
                    description={query.error?.message}
                    actions={[
                      {
                        children: "Retry",
                        onPress: handleRetry,
                        variant: "secondary",
                      },
                    ]}
                  />
                </Cell>
              </TableRow>
            ) : rows.length === 0 ? (
              <TableRow>
                <Cell colSpan={colCount} className="p-2 text-center">
                  {getEmptyContent?.() ?? (
                    <EmptyList
                      illustrationType="juggles_nuts"
                      title="No entries to display"
                      description="Nothing to see here yet. Reset your filters to view results."
                    />
                  )}
                </Cell>
              </TableRow>
            ) : (
              rows.map((row, index) => {
                const group = groups[index];
                const isGroupStart =
                  !!group && groups[index - 1]?.key !== group.key;

                // handle soft selection
                const selectedItem = softSelectionState?.[0];
                const isSoftSelected = !!(
                  isSoftSelectionEnabledForRow?.(row.original) &&
                  selectedItem != null &&
                  table.options.getRowId?.(selectedItem, 0) === row.id
                );
                const handleSoftSelect = isSoftSelectionEnabledForRow?.(
                  row.original
                )
                  ? softSelectionState?.[1]
                  : undefined;

                return (
                  <Fragment key={row.id}>
                    {isGroupStart && (
                      <GroupHeaderRow colCount={colCount}>
                        {group.label}
                      </GroupHeaderRow>
                    )}
                    <InteractiveRow
                      table={table}
                      row={row}
                      isExpanded={row.getIsExpanded()}
                      getLink={getLink}
                      isSoftSelected={isSoftSelected}
                      onSoftSelect={handleSoftSelect}
                      getExpandedContent={getExpandedContent}
                      hasExpandColumn={hasExpandColumn}
                      isTree={isTree}
                      onContextMenu={onRowContextMenu}
                    />
                  </Fragment>
                );
              })
            )}
          </TableBody>
        </Table>
      </div>

      {!disablePagination && <Pagination query={query} table={table} />}
    </div>
  );
}

function GroupHeaderRow({
  colCount,
  children,
}: {
  colCount: number;
  children: ReactNode;
}) {
  return (
    <TableRow className="border-secondary border-b">
      <Cell colSpan={colCount} className="bg-secondary p-0">
        <Typography
          size="xs"
          weight="medium"
          className="text-secondary px-2 py-1.5"
        >
          {children}
        </Typography>
      </Cell>
    </TableRow>
  );
}

type InteractiveRowProps<T extends RowData> = {
  table: TanstackTable<AppTableFeatures, T>;
  row: Row<AppTableFeatures, T>;
  isExpanded: boolean;
  isSoftSelected: boolean;
  onSoftSelect?: Dispatch<T | null>;
  getLink?: (row: T) => LinkProps["href"];
  getExpandedContent?: (row: T) => ReactNode;
  hasExpandColumn: boolean;
  isTree: boolean;
  onContextMenu?: (row: T, event: MouseEvent) => void;
};

function InteractiveRowInner<T extends RowData>({
  table,
  row,
  isExpanded,
  isSoftSelected,
  onSoftSelect,
  getLink,
  getExpandedContent,
  hasExpandColumn,
  isTree,
  onContextMenu,
}: InteractiveRowProps<T>) {
  const withSoftSelection = onSoftSelect !== undefined;
  const canExpand = row.getCanExpand() || !!getExpandedContent;

  const handleRowClick = () => {
    if (canExpand) {
      row.toggleExpanded();
    }
    if (withSoftSelection) {
      onSoftSelect(isSoftSelected ? null : row.original);
    }
  };
  const hidesBottomBorder =
    isExpanded || (row.depth > 0 && row.subRows.length > 0);

  const cellsCount = row.getVisibleCells().length;
  const totalColCount =
    cellsCount +
    (table.options.enableMultiRowSelection ? 1 : 0) +
    (hasExpandColumn ? 1 : 0);

  return (
    <>
      <TableRow
        key={row.id}
        href={getLink?.(row.original)}
        className={clsx(
          "align-center group border-secondary text-primary relative",
          hidesBottomBorder ? "border-b-0" : "border-b last:border-b-0",
          isSoftSelected
            ? "outline-accent bg-accent-tertiary outline-2 -outline-offset-2"
            : getLink || withSoftSelection || canExpand
              ? "hover:bg-primary-hover has-[button:hover]:bg-primary has-[a:hover]:bg-primary has-[[aria-expanded=true]]:bg-primary-hover"
              : "",
          (getLink || withSoftSelection || canExpand) && "cursor-pointer"
        )}
        onAction={handleRowClick}
        onContextMenu={
          onContextMenu && ((event) => onContextMenu(row.original, event))
        }
      >
        {table.options.enableMultiRowSelection && (
          <Cell className="relative w-4 p-2">
            <Checkbox
              slot="selection"
              isDisabled={!row.getCanSelect()}
              isSelected={row.getIsSelected()}
              onChange={() => row.toggleSelected()}
            />
          </Cell>
        )}
        {hasExpandColumn && (
          <Cell className="w-6 py-2 pr-2 pl-4">
            <Icon
              type="chevron-right"
              size="sm"
              className={clsx(
                "text-tertiary shrink-0 transition-transform",
                isExpanded && "!text-accent rotate-90",
                !canExpand && "invisible"
              )}
            />
            {isExpanded && !!getExpandedContent && (
              <div className="bg-accent absolute top-0 left-0 h-full w-0.5" />
            )}
          </Cell>
        )}
        {row.getVisibleCells().map((cell, idx) => {
          const content = flexRender(
            cell.column.columnDef.cell,
            cell.getContext()
          );

          return (
            <Cell
              key={cell.id}
              className={clsx("p-2", cell.column.columnDef.meta?.cellClassName)}
            >
              {isTree && idx === 0 ? (
                <div
                  className="flex items-start gap-2"
                  style={{
                    paddingLeft: `${row.depth * TREE_INDENT_REM}rem`,
                  }}
                >
                  <span className="flex h-5 items-center">
                    <Icon
                      type="chevron-right"
                      size="sm"
                      className={clsx(
                        "text-tertiary shrink-0 transition-transform",
                        isExpanded && "!text-accent rotate-90",
                        !canExpand && "invisible"
                      )}
                    />
                  </span>
                  <div className="min-w-0 flex-1">{content}</div>
                </div>
              ) : (
                content
              )}
            </Cell>
          );
        })}
      </TableRow>
      {isExpanded && getExpandedContent && (
        <TableRow className="border-secondary relative border-b last:border-b-0">
          <Cell
            colSpan={totalColCount}
            className="p-0 pl-0.5"
            data-expanded-row-content
          >
            {getExpandedContent(row.original)}
            <div className="bg-accent absolute top-0 left-0 h-full w-0.5" />
          </Cell>
        </TableRow>
      )}
    </>
  );
}

const InteractiveRow = memo(InteractiveRowInner) as typeof InteractiveRowInner;

// Caret movement / selection keys. A focused text field must be allowed
// to receive them — modifiers included (Shift/Cmd/Ctrl/Alt).
const CARET_NAVIGATION_KEYS = [
  "ArrowLeft",
  "ArrowRight",
  "ArrowUp",
  "ArrowDown",
  "Home",
  "End",
];

/**
 * Keep React Aria Table's grid keyboard navigation from stealing keys that
 * belong to interactive content rendered inside the table.
 *
 * The Table is a grid, and during the capture phase it consumes printable keys
 * (typeahead row-jumping) and Tab (which it uses to move focus out of the whole
 * grid). Two cases need the keys back:
 *
 * - Text fields in cells: printable keys should type, and caret keys
 *   (Arrow/Home/End) should move the cursor, not navigate the grid.
 * - Expanded rows: they host arbitrary content such as forms, so Tab should
 *   move between that content's own focusable controls rather than exit the table.
 *
 * Stopping propagation here leaves the browser's native text entry,
 * caret movement, and Tab focus order intact.
 */
function stopTypeaheadOnTextEntry(e: React.KeyboardEvent) {
  if (
    e.key === "Tab" &&
    e.target instanceof Element &&
    e.target.closest("[data-expanded-row-content]")
  ) {
    e.stopPropagation();
    return;
  }

  if (
    !CARET_NAVIGATION_KEYS.includes(e.key) &&
    (e.key.length !== 1 || e.ctrlKey || e.metaKey || e.altKey)
  ) {
    return;
  }

  const target = e.target;

  if (
    target instanceof HTMLInputElement ||
    target instanceof HTMLTextAreaElement ||
    (target instanceof HTMLElement && target.isContentEditable)
  ) {
    e.stopPropagation();
  }
}
