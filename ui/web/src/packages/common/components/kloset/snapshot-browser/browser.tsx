import {
  Alert,
  Button as UIButton,
  EmptyList,
  Icon,
  IconButton,
  Menu,
  Pagination as UIPagination,
  SegmentedControl,
  Spinner,
  Tooltip,
  Typography,
} from "@plakar-ui/ui/components";
import { useQuery } from "@tanstack/react-query";
import { RowSelectionState } from "@tanstack/react-table";
import {
  createColumnHelper,
  useTable as useReactTable,
} from "@tanstack/react-table";
import clsx from "clsx";
import std_path from "path";
import { ReactNode, useMemo, useState } from "react";
import {
  Button,
  Key,
  LinkProps,
  MenuTrigger,
  Popover,
} from "react-aria-components";

import {
  EnrichedChildEntryType,
  SnapshotType,
  SnapshotVFSQueryOptions,
  useKlosetAPI,
  VFSEntryType,
} from "@plakar-ui/common/api/kloset";
import { isPaginationResponseWithTotal } from "@plakar-ui/common/api/pagination";
import {
  appTableFeatures,
  AppTableFeatures,
  DetailsPanel,
  DirectoryPreview,
  DownloadMenuItem,
  FileIcon,
  FilterPanel,
  PaginatedTable,
  SelectedList,
  SnapshotErrorsAlert,
} from "@plakar-ui/common/components";
import { useLocalStorage } from "@plakar-ui/common/hooks";
import { Main, SidePanelLayout } from "@plakar-ui/common/layouts";

import {
  browseViewCounts,
  BrowseViewId,
  emptyBrowseViews,
  getBrowseView,
  viewerKind,
  suggestBrowseView,
  suggestBrowseViewForEntries,
} from "./browse-views";
import { DOCUMENT_FILTERS, DocumentFilter } from "./document-filters";
import { Gallery, GalleryDensity } from "./gallery";
import { Lightbox } from "./lightbox";
import { MainTitle } from "./main-title";
import { MusicPlayer, MusicView } from "./music";
import { ViewSwitcher } from "./view-switcher";

type FiltersProps = {
  mime?: DocumentFilter;
  filename: string;
};

type BrowseLayout = "list" | "grid";

const DEFAULT_LAYOUTS: Record<BrowseViewId, BrowseLayout> = {
  all: "list",
  photos: "grid",
  videos: "grid",
  music: "list",
  documents: "grid",
};

type FilesBrowserProps = {
  snapshot: SnapshotType;
  source: number;
  vfsEntry: VFSEntryType;
  page: number;
  perPage: number;
  filters: FiltersProps;
  onPageChange: (page: number, per_page: number) => void;
  onPathChange: (path: string) => void;
  onSnapshotChange: (snapshotId: string) => void;
  preview: string;
  onPreviewChange: (value?: string) => void;
  breadcrumb: ReactNode;
  fileBreadcrumb: ReactNode;
  onSearch: (search: string) => void;
  onRemoveFilter: (keys: Set<Key>) => void;
  onApplyFilter: (filters: Record<string, string[] | null>) => void;
  errorPageHref: LinkProps["href"];
  // The menu items an app adds to the snapshot's own. Everything an app alone
  // can answer for lives here -- what a task may do, whether a hold may be
  // placed -- rather than as a prop per item and a second one to gate it.
  actions?: ReactNode;
  onMount?: (mountId: string) => void;
  // The kind of content to browse. Left undefined, the browser picks the view
  // matching what the directory mostly holds; without onViewChange, it keeps
  // the choice to itself.
  view?: BrowseViewId;
  onViewChange?: (view: BrowseViewId) => void;
};

const columnHelper = createColumnHelper<
  AppTableFeatures,
  EnrichedChildEntryType
>();

export function FilesBrowser({
  snapshot,
  source,
  vfsEntry,
  page,
  perPage,
  filters,
  onPageChange,
  onPathChange,
  onSnapshotChange,
  preview,
  onPreviewChange,
  breadcrumb,
  fileBreadcrumb,
  onSearch,
  onRemoveFilter,
  onApplyFilter,
  errorPageHref,
  actions,
  onMount,
  view,
  onViewChange,
}: FilesBrowserProps) {
  const [localView, setLocalView] = useState<BrowseViewId>();
  const [playingPath, setPlayingPath] = useState<string | null>(null);
  const listing = `${vfsEntry.path}:${page}`;
  const [playingListing, setPlayingListing] = useState(listing);
  if (playingListing !== listing) {
    setPlayingListing(listing);
    setPlayingPath(null);
  }
  const [lightbox, setLightbox] = useState<{
    path: string;
    followsPreview: boolean;
  } | null>(null);
  const explicitView = view ?? localView;
  const changeView = (next: BrowseViewId) => {
    setLightbox(null);
    setPlayingPath(null);
    if (onViewChange) onViewChange(next);
    else setLocalView(next);
  };

  const [layouts, setLayouts] = useLocalStorage<
    Partial<Record<BrowseViewId, BrowseLayout>>
  >("browserLayouts", {});
  const [density, setDensity] = useLocalStorage<GalleryDensity>(
    "browserDensity",
    "medium"
  );

  // Without a view picked, a directory opens in the view fitting what it holds
  // and a search lists everything it finds, to be shown in the view fitting
  // its results once they are in.
  const hasFilters = Object.values(filters).some((filter) => filter);
  const queryView =
    explicitView ?? (hasFilters ? "all" : suggestBrowseView(vfsEntry));
  const queryDef = getBrowseView(queryView);
  const isSearching = Boolean(filters.filename);

  const search = queryDef.mimes
    ? {
        contentTypes:
          queryView === "documents" && filters.mime
            ? filters.mime.contentTypes
            : queryDef.mimes,
        filename: filters.filename,
        recursive: (vfsEntry.summary?.directory.directories ?? 0) > 0,
      }
    : isSearching
      ? { filename: filters.filename }
      : undefined;

  const { api, repository } = useKlosetAPI();
  const query = useQuery({
    ...SnapshotVFSQueryOptions(api, {
      repository,
      snapshot: snapshot.identifier.Value,
      source,
      vfsEntry,
      page,
      perPage,
      search,
    }),
    // Keep showing the previous results while the next ones load, but not
    // those of another kind of content: filtered by this view, they would
    // read as an empty directory until the new results are in.
    placeholderData: (previous, previousQuery) =>
      previousQuery &&
      String(previousQuery.queryKey[2].search?.contentTypes) ===
        String(search?.contentTypes)
        ? previous
        : undefined,
  });

  const activeView =
    explicitView ??
    (isSearching
      ? suggestBrowseViewForEntries(query.data?.items ?? [])
      : queryView);
  const isAutoView = explicitView === undefined;
  const viewDef = getBrowseView(activeView);
  const layout = layouts[activeView] ?? DEFAULT_LAYOUTS[activeView];
  const isListView = layout === "list";
  const showsTable = isListView && activeView !== "music";
  const changeLayout = (next: BrowseLayout) => {
    setPlayingPath(null);
    setLayouts((previous) => ({ ...previous, [activeView]: next }));
  };
  const shownNoun =
    activeView === "documents" && filters.mime
      ? filters.mime.noun
      : viewDef.name.toLowerCase();

  const columns = useMemo(
    () =>
      columnHelper.columns([
        columnHelper.accessor("file_info.name", {
          header: () => "Name",
          cell: ({ row: { original: child } }) => {
            const { mode, name } = child.file_info;
            const isDir = mode.isDir();
            const isFile = mode.isRegular();
            const isLink = mode.isSymlink();
            const errors = child.summary?.Total.errors ?? 0;
            const showError = isDir && name !== ".." && errors > 0;

            // When searching for files, the files returned are flat and might have a
            // different parent path than the current vfsEntry.
            const relativePath = std_path.relative(
              vfsEntry.path,
              child.parent_path
            );

            return (
              <div className="grid grid-cols-[auto_1fr] items-center gap-1">
                <div className="flex h-6 items-center">
                  {isDir && (
                    <>
                      {showError && (
                        <Tooltip
                          label={`${errors} errors in this directory`}
                          offset={4}
                        >
                          <Icon
                            type="md-error"
                            size="xl"
                            className="text-error"
                            role="button"
                          />
                        </Tooltip>
                      )}
                      <Icon type="md-folder" size="xl" />
                    </>
                  )}
                  {isFile && <FileIcon contentType={child.contentType} />}
                  {isLink && <Icon type="md-link" size="xl" />}
                </div>

                {isDir ? (
                  <Typography
                    variant="body"
                    size="sm"
                    className={clsx(
                      "cursor-pointer text-left break-all hover:underline"
                    )}
                    as={Button}
                    onPress={() => {
                      if (isDir) {
                        onPathChange(child.path);
                      }
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

                {relativePath && query.data?.isSearching && (
                  <Typography
                    variant="body"
                    size="xs"
                    className="text-tertiary col-start-2 w-fit cursor-pointer text-left break-all hover:underline"
                    as={Button}
                    onPress={() => {
                      onPathChange(child.parent_path);
                    }}
                  >
                    {relativePath}
                  </Typography>
                )}
              </div>
            );
          },
        }),

        columnHelper.display({
          id: "size",
          meta: {
            cellClassName: "md:w-[15%]",
          },
          header: () => "Size",
          cell: ({ row: { original: child } }) => {
            // Parent‐dir special case
            if (child.file_info.name === "..") {
              return;
            }

            // Pick the right size object
            const isDir = child.file_info.mode.isDir();
            const sizeObj = isDir
              ? child.summary?.Total.size
              : child.file_info.size;

            // Fallbacks in case size is missing
            const bytes = sizeObj?.Value?.toString() ?? "0";
            const pretty = sizeObj?.Pretty;
            const weightPct = child.weight.directory;
            const suffix = weightPct > 1 ? ` (${weightPct.toFixed(0)}%)` : "";

            return (
              <Tooltip label={`${bytes} bytes`} offset={4}>
                <span className="inline-block" role="button">
                  <Typography
                    variant="body"
                    size="sm"
                    className="text-secondary w-fit whitespace-nowrap"
                  >
                    {pretty}
                    {suffix}
                  </Typography>
                </span>
              </Tooltip>
            );
          },
        }),

        columnHelper.accessor("file_info.mod_time", {
          meta: {
            cellClassName: "md:w-[20%]",
          },
          header: () => "Modified",
          cell: (info) => {
            const modTime = info.getValue();

            // Hide last modified the .. folder
            if (
              info.row.original.file_info.name === ".." ||
              !modTime.toHumanString()
            ) {
              return;
            }

            return (
              <Tooltip label={modTime.Value} offset={4}>
                <span className="inline-block" role="button">
                  <Typography
                    variant="body"
                    size="sm"
                    className="text-secondary"
                  >
                    {modTime.toHumanString()}
                  </Typography>
                </span>
              </Tooltip>
            );
          },
        }),
      ]),
    [onPathChange, query.data?.isSearching, vfsEntry.path]
  );

  const paginationState = {
    pageIndex: page,
    pageSize: perPage,
  };

  const [selected, setSelected] = useState<EnrichedChildEntryType[]>([]);

  // Build a RowSelectionState object from the selected entries. This object
  // contains the names of the selected entries as keys, and "true" as
  // value.
  const rowSelection = selected.reduce((acc, item) => {
    acc[item.path] = true;
    return acc;
  }, {} as RowSelectionState);

  // The search endpoint matches content types loosely when it does not recurse
  // (documents ask for all of application/), so views filter what comes back.
  const items = (query.data?.items ?? []).filter(
    (item) =>
      activeView === "all" ||
      (!item.file_info.mode.isDir() && viewDef.matches(item.contentType))
  );
  const table = useReactTable({
    features: appTableFeatures,
    data: activeView === "all" ? (query.data?.items ?? []) : items,
    rowCount:
      query.data && isPaginationResponseWithTotal(query.data)
        ? query.data.total
        : undefined,
    columns,
    state: {
      pagination: paginationState,
      rowSelection,
    },

    // Disable row selection for the ".." entry.
    enableRowSelection: (row) => {
      return row.original.file_info.name !== "..";
    },

    onRowSelectionChange: (updaterFn) => {
      // newSelection is an object with the checksums of the selected entries as keys, and "true" as value.
      const newSelection =
        typeof updaterFn === "function" ? updaterFn(rowSelection) : updaterFn;

      // Build the new selected items array. For each entry in newSelection:
      // * Try to find it in the current selected array. If it's found, return
      // the item. The item might be from the current page or from the previous
      // pages.
      // * If the item is not found, it means it's a new item that was selected
      // on the current page. In this case, try to find it in the current page
      // data.

      setSelected((prev) =>
        Object.keys(newSelection)
          .map((fullpath) => {
            return (
              prev.find((item) => fullpath === item.path) ||
              query.data?.items.find((item) => fullpath === item.path)
            );
          })
          .filter((v) => v !== undefined)
      );
    },
    getRowId: (row) => row.path,
    manualPagination: true,
    onPaginationChange: (pagination) => {
      const state =
        typeof pagination === "function"
          ? pagination(paginationState)
          : pagination;

      onPageChange(state.pageIndex, state.pageSize);
    },
  });

  const currentIndex = items.findIndex((item) => item.path === preview);
  const previewEntry = currentIndex === -1 ? undefined : items[currentIndex];

  // if currentIndex is not found, rest contains all elements starting with the first
  const rest = items.slice(currentIndex + 1);
  const before =
    currentIndex === -1 ? [] : items.slice(0, currentIndex).reverse();

  const nextOffset = rest.findIndex((item) => !item.file_info.mode.isDir());
  const prevOffset = before.findIndex((item) => !item.file_info.mode.isDir());

  const isNextDisabled = nextOffset === -1;
  const isPreviousDisabled = prevOffset === -1;

  const handleNextFile = () => {
    if (isNextDisabled) return;
    const nextItem = items[currentIndex + 1 + nextOffset];
    if (nextItem) onPreviewChange(nextItem.path);
  };

  const handlePreviousFile = () => {
    if (isPreviousDisabled) return;
    const prevItem = items[currentIndex - prevOffset - 1];
    if (prevItem) onPreviewChange(prevItem.path);
  };

  const selectedPaths = new Set(selected.map((item) => item.path));
  const toggleSelected = (entry: EnrichedChildEntryType) =>
    setSelected((prev) =>
      prev.some((item) => item.path === entry.path)
        ? prev.filter((item) => item.path !== entry.path)
        : [...prev, entry]
    );

  const lightboxEntry = lightbox
    ? items.find((item) => item.path === lightbox.path)
    : undefined;
  const lightboxKind = lightboxEntry && viewerKind(lightboxEntry.contentType);
  const lightboxItems = lightboxKind
    ? items.filter(
        (item) =>
          !item.file_info.mode.isDir() &&
          viewerKind(item.contentType) === lightboxKind
      )
    : [];
  const lightboxIndex = lightboxItems.findIndex(
    (item) => item.path === lightbox?.path
  );
  if (lightbox && lightboxIndex === -1) {
    setLightbox(null);
  }
  const openEntry = (
    entry: EnrichedChildEntryType,
    { followPreview = false } = {}
  ) => {
    if (viewerKind(entry.contentType) === null) {
      onPreviewChange(entry.path);
      return;
    }
    setPlayingPath(null);
    setLightbox({ path: entry.path, followsPreview: followPreview });
  };
  const changeLightboxIndex = (index: number) => {
    const { path } = lightboxItems[index];
    setLightbox((current) => current && { ...current, path });
    if (lightbox?.followsPreview) {
      onPreviewChange(path);
    }
  };

  // In a view, the view is the type filter: only the name search remains.
  const filterBar = (
    <FilterPanel>
      <FilterPanel.Search
        value={filters.filename}
        onSearch={onSearch}
        aria-label="Search files"
        placeholder={
          activeView === "all"
            ? "Search by name or *.txt"
            : `Search ${viewDef.name.toLowerCase()} by name`
        }
      />
      {activeView === "documents" && (
        <FilterPanel.Groups>
          <FilterPanel.Group label="Format">
            <SegmentedControl
              variant="accent"
              ariaLabel="Document format"
              items={[
                { id: "all", label: "All" },
                ...DOCUMENT_FILTERS.map(({ id, name, icon }) => ({
                  id,
                  label: name,
                  icon,
                })),
              ]}
              selectedId={filters.mime?.id ?? "all"}
              onSelect={(id) =>
                id === "all"
                  ? onRemoveFilter(new Set(["mime"]))
                  : onApplyFilter({ mime: [id] })
              }
            />
          </FilterPanel.Group>
        </FilterPanel.Groups>
      )}
    </FilterPanel>
  );
  // The results a view picked from a search leaves out.
  const hiddenCount =
    hasFilters && activeView !== "all"
      ? (query.data?.items.length ?? 0) - items.length
      : 0;

  const hasNextPage = Boolean(
    query.data &&
    (isPaginationResponseWithTotal(query.data)
      ? (page + 1) * perPage < query.data.total
      : query.data.has_next)
  );

  return (
    <SidePanelLayout
      main={
        <Main
          title={<MainTitle snapshot={snapshot} source={source} />}
          isLoading={query.isFetching}
          breadcrumb={breadcrumb}
          controls={
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
                    paths={[vfsEntry.path]}
                  />
                  {actions}
                </Menu>
              </Popover>
            </MenuTrigger>
          }
        >
          <div className="flex flex-col gap-4">
            {fileBreadcrumb}

            <div className={clsx(selected.length > 0 && "mb-18")}>
              <div className="flex flex-col gap-4">
                <DirectoryPreview
                  snapshot={snapshot}
                  source={source}
                  childEntry={vfsEntry}
                  onSnapshotChange={onSnapshotChange}
                />

                {vfsEntry.summary && vfsEntry.summary.Total.errors > 0 && (
                  <SnapshotErrorsAlert
                    href={errorPageHref}
                    vfsEntry={vfsEntry}
                  />
                )}

                <div className="flex flex-col gap-3">
                  <div className="border-secondary flex flex-col gap-2 md:flex-row md:items-center md:justify-between md:gap-3 md:border-b">
                    <ViewSwitcher
                      className="border-secondary border-b md:border-b-0"
                      view={activeView}
                      counts={browseViewCounts(vfsEntry)}
                      emptyViews={emptyBrowseViews(vfsEntry)}
                      onChange={changeView}
                    />
                    <div className="flex items-center justify-between gap-2 md:justify-end">
                      <SegmentedControl
                        ariaLabel="Layout"
                        hideLabels
                        items={[
                          { id: "list", label: "List", icon: "list" },
                          {
                            id: "grid",
                            label: "Grid",
                            icon: "grid",
                          },
                        ]}
                        selectedId={layout}
                        onSelect={(id) =>
                          changeLayout(id === "grid" ? "grid" : "list")
                        }
                      />
                    </div>
                  </div>

                  {isAutoView && activeView !== "all" && (
                    <Alert
                      variant="brand"
                      icon="wand-magic-sparkles"
                      title={`Showing the ${viewDef.name} view`}
                      actions={
                        <UIButton
                          size="sm"
                          variant="accent-quaternary"
                          onPress={() => changeView("all")}
                        >
                          {hasFilters
                            ? "Show all results"
                            : "Browse this folder"}
                        </UIButton>
                      }
                    >
                      {hasFilters
                        ? `These results are mostly ${viewDef.name.toLowerCase()}.`
                        : `This folder mostly holds ${viewDef.name.toLowerCase()}.`}
                      {hiddenCount > 0 &&
                        ` ${hiddenCount} other ${hiddenCount === 1 ? "result is" : "results are"} in Browse.`}
                    </Alert>
                  )}
                </div>

                {!showsTable && (
                  <div className="flex flex-col gap-4">
                    {filterBar}
                    {query.isPending ? (
                      <Spinner size="2xl" className="p-10" />
                    ) : query.isError ? (
                      <Alert variant="error" title="Unable to list this folder">
                        {query.error.message}
                      </Alert>
                    ) : items.length === 0 ? (
                      <EmptyList
                        size="sm"
                        title={
                          activeView === "all"
                            ? isSearching
                              ? "Nothing matches this search"
                              : "This folder is empty"
                            : isSearching
                              ? `No ${shownNoun} match this search`
                              : `No ${shownNoun} here`
                        }
                        description={
                          isSearching
                            ? "Try another name."
                            : activeView === "all"
                              ? "It holds no files or folders."
                              : `Neither this folder nor its subfolders hold any ${shownNoun}.`
                        }
                      />
                    ) : activeView === "music" && isListView ? (
                      <MusicView
                        snapshot={snapshot.identifier.Value}
                        source={source}
                        tracks={items}
                        basePath={vfsEntry.path}
                        selected={selectedPaths}
                        onToggleSelected={toggleSelected}
                        playingPath={playingPath}
                        onPlay={setPlayingPath}
                      />
                    ) : (
                      <Gallery
                        snapshot={snapshot.identifier.Value}
                        source={source}
                        view={activeView}
                        entries={items}
                        basePath={vfsEntry.path}
                        density={density}
                        onDensityChange={setDensity}
                        selected={selectedPaths}
                        onToggleSelected={toggleSelected}
                        onOpen={openEntry}
                        onOpenDirectory={onPathChange}
                      />
                    )}
                    <UIPagination
                      currentPage={page}
                      onPrev={
                        page > 0
                          ? () => onPageChange(page - 1, perPage)
                          : undefined
                      }
                      onNext={
                        hasNextPage
                          ? () => onPageChange(page + 1, perPage)
                          : undefined
                      }
                      onPageSizeChange={(size) => onPageChange(0, size)}
                      total={{
                        count:
                          query.data &&
                          isPaginationResponseWithTotal(query.data)
                            ? query.data.total
                            : undefined,
                        perPage,
                      }}
                    />
                  </div>
                )}

                {showsTable && (
                  <PaginatedTable<EnrichedChildEntryType>
                    query={query}
                    table={table}
                    softSelectionState={[
                      previewEntry ?? null,
                      (entry) => {
                        if (entry) {
                          onPreviewChange(entry.path);
                        }
                      },
                    ]}
                    isSoftSelectionEnabledForRow={(row) =>
                      !row.file_info.mode.isDir()
                    }
                    filters={filterBar}
                    ariaLabel="Restore point files table"
                    isLoading={query.isFetching}
                  />
                )}
              </div>

              <div className="pointer-events-none absolute inset-x-0 bottom-[calc(var(--pagination-height)+0.75rem)] flex flex-col items-center gap-2 px-3 *:pointer-events-auto">
                {activeView === "music" && isListView && (
                  <MusicPlayer
                    snapshot={snapshot.identifier.Value}
                    source={source}
                    tracks={items}
                    playingPath={playingPath}
                    onPlay={setPlayingPath}
                  />
                )}
                {selected.length > 0 && (
                  <SelectedList
                    snapshot={snapshot}
                    childEntries={selected}
                    removeSelection={() => setSelected([])}
                  />
                )}
              </div>
            </div>
          </div>
          {lightboxIndex >= 0 && (
            <Lightbox
              snapshot={snapshot}
              source={source}
              items={lightboxItems}
              index={lightboxIndex}
              onIndexChange={changeLightboxIndex}
              onClose={() => setLightbox(null)}
            />
          )}
        </Main>
      }
      sidebar={
        preview &&
        lightboxIndex < 0 && (
          <DetailsPanel
            snapshot={snapshot}
            source={source}
            path={preview}
            onSnapshotChange={onSnapshotChange}
            onMount={onMount}
            onExpand={
              previewEntry && viewerKind(previewEntry.contentType)
                ? () => openEntry(previewEntry, { followPreview: true })
                : undefined
            }
          />
        )
      }
      controls={[
        {
          icon: "chevron-down",
          tooltip: "Next file",
          isDisabled: isNextDisabled,
          onPress: handleNextFile,
        },
        {
          icon: "chevron-up",
          tooltip: "Previous file",
          isDisabled: isPreviousDisabled,
          onPress: handlePreviousFile,
        },
      ]}
      onCollapse={() => onPreviewChange(undefined)}
    />
  );
}
