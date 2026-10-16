import { Icon, Typography } from "@plakar-ui/ui/components";
import { UseQueryResult } from "@tanstack/react-query";
import {
  createColumnHelper,
  useTable as useReactTable,
} from "@tanstack/react-table";
import clsx from "clsx";
import std_path from "path";
import { useCallback, useMemo } from "react";
import { Button as RAButton } from "react-aria-components";

import { MountFileEntry } from "@plakar-ui/common/api/kloset";
import {
  appTableFeatures,
  AppTableFeatures,
  FileIcon,
  PaginatedTable,
} from "@plakar-ui/common/components";

const columnHelper = createColumnHelper<AppTableFeatures, MountFileEntry>();

type MountBrowserProps = {
  query: UseQueryResult<{ items: MountFileEntry[]; total: number }>;
  path: string;
  preview?: string;
  onPathChange: (path: string) => void;
  onPreviewChange: (preview?: string) => void;
};

export function MountBrowser({
  query,
  path,
  preview,
  onPathChange,
  onPreviewChange,
}: MountBrowserProps) {
  const columns = useMemo(
    () =>
      columnHelper.columns([
        columnHelper.accessor("name", {
          header: () => "Name",
          cell: (info) => {
            const name = info.getValue();
            const chmod = info.row.original.mode;

            return (
              <div className="grid grid-cols-[auto_1fr] items-center gap-1">
                <div className="flex h-6 items-center">
                  {chmod.isDir() && (
                    <>
                      <Icon type="md-folder" size="xl" />
                    </>
                  )}
                  {chmod.isRegular() && <FileIcon contentType="text" />}
                  {chmod.isSymlink() && <Icon type="md-link" size="xl" />}
                </div>

                {chmod.isDir() ? (
                  <Typography
                    variant="body"
                    size="sm"
                    className={clsx(
                      "cursor-pointer text-left break-all hover:underline"
                    )}
                    as={RAButton}
                    onPress={() => {
                      onPathChange(std_path.join(path, name));
                    }}
                  >
                    {name}
                  </Typography>
                ) : (
                  <Typography
                    variant="body"
                    size="sm"
                    className="break-all"
                    as="span"
                  >
                    {name}
                  </Typography>
                )}
              </div>
            );
          },
        }),
        columnHelper.accessor("size", {
          header: () => "Size",
          cell: (info) => {
            return (
              <Typography variant="body" size="sm" className="text-secondary">
                {info.getValue().Pretty}
              </Typography>
            );
          },
        }),
        columnHelper.accessor("mode", {
          header: () => "Mode",
          cell: (info) => {
            return (
              <Typography variant="body" size="sm" className="text-secondary">
                {info.getValue().toString()}
              </Typography>
            );
          },
        }),
        columnHelper.accessor("mod_time", {
          header: () => "Last Modified",
          cell: (info) => {
            return (
              <Typography variant="body" size="sm" className="text-secondary">
                {info.getValue().toHumanString()}
              </Typography>
            );
          },
        }),
      ]),
    [onPathChange, path]
  );

  const paginationState = {
    pageIndex: 0,
    pageSize: query.data?.total ?? 0,
  };

  const table = useReactTable({
    features: appTableFeatures,
    data: query.data?.items ?? [],
    rowCount: query.data?.total ?? 0,
    columns,
    state: {
      pagination: paginationState,
    },
    enableRowSelection: false,
    enableMultiRowSelection: false,
    getRowId: (row) => row.name,
    manualPagination: true,
  });

  const selectedEntry =
    query.data?.items.find(
      (item) => std_path.join(path, item.name) === preview
    ) ?? null;

  const setSoftSelection = useCallback(
    (entry: MountFileEntry | null) =>
      onPreviewChange(std_path.join(path, entry?.name ?? "")),
    [onPreviewChange, path]
  );

  return (
    <PaginatedTable<MountFileEntry>
      query={query}
      table={table}
      softSelectionState={[selectedEntry, setSoftSelection]}
      isSoftSelectionEnabledForRow={(row) => !row.mode.isDir()}
      disablePagination
    />
  );
}
