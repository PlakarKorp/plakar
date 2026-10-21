import {
  RepositoryImporterTypesQueryOptions,
  RepositoryInfoQueryOptions,
  RepositorySnapshotsQueryOptions,
  SnapshotType,
  useKlosetAPI,
} from "@plakar-ui/common/api/kloset";
import {
  AppTableFeatures,
  appTableFeatures,
  DownloadMenuItem,
  PaginatedTable,
} from "@plakar-ui/common/components";
import { Main } from "@plakar-ui/common/layouts";
import { createPaginationSearchSchema } from "@plakar-ui/common/utils";
import {
  Alert,
  Errors,
  ExpandableList,
  IconButton,
  Id,
  Menu,
  Select,
  Tag,
  Typography,
} from "@plakar-ui/ui/components";
import { useQuery, useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { RowSelectionState } from "@tanstack/react-table";
import {
  createColumnHelper,
  useTable as useReactTable,
} from "@tanstack/react-table";
import { useState } from "react";
import { MenuTrigger, Popover } from "react-aria-components";
import { z } from "zod";

export const Route = createFileRoute("/_layout/snapshots/")({
  validateSearch: createPaginationSearchSchema().extend({
    importer: z.string().catch("").default(""),
  }),
  component: Component,
});

const NON_BROWSABLE_ERROR_MESSAGE =
  "This repository uses a non-browsable storage, restore points cannot be inspected.";

function Component() {
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});

  const navigate = Route.useNavigate();
  const search = Route.useSearch();
  const ctx = Route.useRouteContext();
  const repositoryInfoQuery = useSuspenseQuery(
    RepositoryInfoQueryOptions(ctx.klosetAPI, { repository: undefined })
  );
  const browsable = repositoryInfoQuery.data.item.browsable;

  const query = useQuery(
    RepositorySnapshotsQueryOptions(ctx.klosetAPI, {
      repository: undefined,
      importer: search.importer,
      pagination: {
        page: search.page,
        perPage: search.per_page,
      },
    })
  );

  const columnHelper = createColumnHelper<AppTableFeatures, SnapshotType>();

  const columns = columnHelper.columns([
    columnHelper.display({
      id: "identifier",
      header: () => "ID",
      cell: (info) => {
        const row = info.row.original;
        const id = row.identifier.Value;
        const errorLength = row.total.errors;
        return (
          <div className="flex flex-col gap-1">
            <Id value={id} />
            <Errors length={errorLength} />
          </div>
        );
      },
    }),
    columnHelper.accessor("timestamp", {
      header: () => "Date",
      cell: (info) => {
        const date = info.getValue();
        return (
          <div className="flex flex-col gap-1">
            <Typography
              variant="body"
              size="md"
              className="text-primary whitespace-nowrap"
            >
              {date.relativeToNow()}
            </Typography>
            <Typography variant="body" size="sm" className="text-secondary">
              {date.toHumanString()}
            </Typography>
          </div>
        );
      },
    }),
    columnHelper.accessor("duration", {
      header: () => "Duration",
      cell: (info) => {
        const duration = info.getValue();
        return (
          <div className="flex flex-col gap-1 whitespace-nowrap">
            <Typography variant="body" size="md" className="text-primary">
              {duration.toShortString()}
            </Typography>
            <Typography variant="body" size="sm" className="text-secondary">
              Duration
            </Typography>
          </div>
        );
      },
    }),
    columnHelper.display({
      id: "importer-type",
      header: () => "Source",
      meta: {
        cellClassName: "w-[60%] max-w-0 min-w-80",
      },
      cell: ({ row: { original: child } }) => {
        const tags = child.tags.filter((v) => v.trim());
        return (
          <div className="flex w-full min-w-0 items-baseline gap-4">
            <div className="min-w-0 flex-1">
              <ExpandableList
                items={child.sources.map((source) => ({
                  tag: source.importer.type,
                  value: source.importer.origin,
                  subValue: source.importer.directory.toHumanString(),
                }))}
              />
            </div>
            {tags.length > 0 && (
              <span className="flex max-w-[40%] min-w-0 flex-initial flex-wrap items-start gap-1">
                {tags
                  .filter((v) => v.trim())
                  .map((tag) => (
                    <Tag
                      key={tag}
                      size="sm"
                      variant="blue_borderless"
                      radius="sqrt"
                    >
                      {tag}
                    </Tag>
                  ))}
              </span>
            )}
          </div>
        );
      },
    }),
    columnHelper.accessor("total", {
      id: "summary",
      header: () => "Stats",
      cell: (info) => {
        const total = info.getValue();
        return (
          <div className="flex flex-col gap-1 whitespace-nowrap">
            <Typography variant="body" size="md" className="text-primary">
              {total.size.Pretty}
            </Typography>
            <Typography variant="body" size="sm" className="text-secondary">
              {total.files} {total.files === 1 ? "item" : "items"}
            </Typography>
          </div>
        );
      },
    }),
    columnHelper.display({
      id: "actions",
      header: () => "Actions",
      cell: ({ row: { original: snapshot } }) => {
        return (
          <MenuTrigger>
            <IconButton
              icon="ellipsis"
              aria-label="More"
              variant="tertiary"
              className="cursor-pointer"
            />
            <Popover placement="bottom end" offset={4}>
              <Menu>
                <DownloadMenuItem
                  snapshotId={snapshot.identifier.Value}
                  paths={[snapshot.sources[0].importer.directory.Value]}
                />
                <Menu.Item
                  leftIcon="not-equal"
                  label="Compare with previous"
                  href={{
                    to: "/compare",
                    search: { b: snapshot.identifier.Value },
                  }}
                />
              </Menu>
            </Popover>
          </MenuTrigger>
        );
      },
    }),
  ]);

  const paginationState = {
    pageIndex: search.page,
    pageSize: search.per_page,
  };

  const table = useReactTable({
    features: appTableFeatures,
    data: query.data?.items ?? [],
    rowCount: query.data?.total,
    columns,
    state: {
      pagination: paginationState,
      rowSelection,
    },
    onRowSelectionChange: setRowSelection,
    enableMultiRowSelection: false, // Only allow single row selection
    onPaginationChange: (pagination) => {
      const state =
        typeof pagination === "function"
          ? pagination(paginationState)
          : pagination;
      navigate({
        search: (prev) => ({
          ...prev,
          page: prev.per_page !== state.pageSize ? 0 : state.pageIndex,
          per_page: state.pageSize,
        }),
      });
    },
    manualPagination: true,
  });

  return (
    <Main
      icon="folder-tree"
      iconBadgeType="store"
      title="Restore points"
      description="Browse backups and restore files."
    >
      <div className="flex flex-col gap-4">
        {!browsable && (
          <Alert
            title="This restore point repository is not browsable."
            variant="warning"
          >
            {NON_BROWSABLE_ERROR_MESSAGE}
          </Alert>
        )}

        <PaginatedTable<SnapshotType>
          query={query}
          table={table}
          filters={<ImporterTypeFilter />}
          variant="cards"
          skeleton="two-rows"
          getLink={
            !browsable
              ? undefined
              : (snapshot) => ({
                  to: "/snapshots/$snapshotId/$source/browse",
                  params: {
                    snapshotId: snapshot.identifier.Value,
                    source: 0,
                  },
                  search: {
                    path: snapshot.sources[0].importer.directory.Value,
                  },
                })
          }
          ariaLabel="Restore points table"
        />
      </div>
    </Main>
  );
}

function ImporterTypeFilter() {
  const search = Route.useSearch();
  const navigate = Route.useNavigate();
  const { api } = useKlosetAPI();

  const { data, isPending } = useQuery(
    RepositoryImporterTypesQueryOptions(api, { repository: undefined })
  );

  const options =
    data?.items.map((entry) => ({ id: entry.name, name: entry.name })) || [];

  return (
    <Select
      aria-label="Filter by source type"
      placeholder="All source types"
      value={search.importer || null}
      onChange={(key) =>
        navigate({
          search: (prev) => ({
            ...prev,
            importer: key ? String(key) : "",
            page: 0,
          }),
        })
      }
      isClearable
      isPending={isPending}
      // A type from the URL stays clearable even when none can be listed.
      isDisabled={!search.importer && options.length === 0}
      isInvalid={
        !!search.importer &&
        !isPending &&
        !options.some((o) => o.id === search.importer)
      }
      selectedFallbackOption={
        search.importer
          ? { id: search.importer, name: search.importer }
          : undefined
      }
      className="ml-auto min-w-60 max-sm:w-full"
    >
      <Select.ListBox items={options}>
        {(item) => <Select.Item {...item} />}
      </Select.ListBox>
    </Select>
  );
}
