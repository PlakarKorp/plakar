import {
  GetSnapshotPathQueryOptions,
  GetSnapshotQueryOptions,
} from "@plakar-ui/common/api/kloset";
import {
  BROWSE_VIEW_IDS,
  BrowseViewId,
  Breadcrumbs,
  FilesBrowser,
  MIME_FILTERS,
} from "@plakar-ui/common/components";
import { createPaginationSearchSchema } from "@plakar-ui/common/utils";
import { Menu } from "@plakar-ui/ui/components";
import { createFileRoute, redirect } from "@tanstack/react-router";
import std_path from "path";
import { useCallback } from "react";
import { Key } from "react-aria-components";
import { z } from "zod";

import {
  breadcrumbsForPath,
  breadcrumbsForSource,
} from "~/utils/breadcrumbs-for-path";

export const Route = createFileRoute(
  "/_layout/snapshots/$snapshotId/$source/browse"
)({
  params: {
    parse: (raw) => ({
      snapshotId: raw.snapshotId,
      source: z.coerce
        .number()
        .nonnegative()
        .catch(0)
        .default(0)
        .parse(raw.source),
    }),
    stringify: (p) => ({
      snapshotId: p.snapshotId,
      source: p.source.toString(),
    }),
  },

  validateSearch: createPaginationSearchSchema().extend({
    path: z.string().min(1).catch("/").default("/"),
    mime: z
      .string()
      .transform((value) => MIME_FILTERS.find((m) => m.id === value)?.id ?? "")
      .catch("")
      .default(""),
    filename: z.string().catch("").default(""),
    preview: z.string().catch("").default(""),
    // Left out, the browser picks the view that fits the directory.
    view: z.enum(BROWSE_VIEW_IDS).optional().catch(undefined),
  }),

  loaderDeps: ({ search }) => {
    return { path: search.path };
  },

  loader: async ({ context, deps, params }) => {
    // For now, the API doesn't support browsing other sources than 0.
    if (params.source !== 0) {
      throw redirect({
        to: "/snapshots/$snapshotId/$source/browse",
        params: {
          snapshotId: params.snapshotId,
          source: 0,
        },
      });
    }

    const [getSnapshotResponse, getSnapshotPathResponse] = await Promise.all([
      context.queryClient.ensureQueryData(
        GetSnapshotQueryOptions(context.klosetAPI, {
          repository: undefined,
          snapshot: params.snapshotId,
        })
      ),
      context.queryClient.ensureQueryData(
        GetSnapshotPathQueryOptions(context.klosetAPI, {
          repository: undefined,
          snapshot: params.snapshotId,
          source: params.source,
          path: deps.path,
        })
      ),
    ]);

    if (!getSnapshotPathResponse.item.file_info.mode.isDir()) {
      throw redirect({
        to: "/snapshots/$snapshotId/$source/browse",
        params: { snapshotId: params.snapshotId, source: params.source },
        search: {
          path: std_path.dirname(deps.path),
          page: 0,
        },
      });
    }

    return {
      getSnapshotResponse,
      getSnapshotPathResponse,
    };
  },

  component: Component,
});

function Component() {
  const params = Route.useParams();
  const search = Route.useSearch();
  const data = Route.useLoaderData();
  const navigate = Route.useNavigate();
  const selectedMimeFilter = MIME_FILTERS.find(
    (mime) => mime.id === search.mime
  );
  const snapshot = data.getSnapshotResponse.item;
  const vfsEntry = data.getSnapshotPathResponse.item;

  const onPathChange = useCallback(
    (path: string) =>
      navigate({
        params: { snapshotId: params.snapshotId },
        search: (prev) => ({
          ...prev,
          path,
          page: 0,
          view: undefined,
          filename: undefined,
          mime: undefined,
        }),
      }),
    [navigate, params.snapshotId]
  );

  const onPageChange = useCallback(
    (page: number, per_page: number) =>
      navigate({
        search: (prev) => ({
          ...prev,
          page: prev.per_page !== per_page ? 0 : page,
          per_page,
        }),
      }),
    [navigate]
  );

  const onSnapshotChange = useCallback(
    (snapshotId: string) => {
      navigate({
        params: { snapshotId },
        search: (prev) => ({
          ...prev,
          page: 0,
        }),
      });
    },
    [navigate]
  );

  const onPreviewChange = useCallback(
    (preview?: string) => {
      navigate({
        params: { snapshotId: params.snapshotId },
        search: (prev) => ({
          ...prev,
          preview,
        }),
        replace: true,
      });
    },
    [navigate, params.snapshotId]
  );

  const onViewChange = useCallback(
    (view: BrowseViewId) =>
      navigate({
        search: (prev) => ({
          ...prev,
          view,
          page: 0,
          preview: "",
          mime: undefined,
        }),
      }),
    [navigate]
  );

  const handleSearch = useCallback(
    (newFilename: string) =>
      navigate({
        search: (prev) => ({
          ...prev,
          page: 0,
          filename: newFilename || undefined,
        }),
      }),
    [navigate]
  );

  const handleRemoveFilter = useCallback(
    (keys: Set<Key>) => {
      navigate({
        search: (prev) => ({
          ...prev,
          mime: keys.has("mime") ? "" : prev.mime,
          page: 0,
        }),
      });
    },
    [navigate]
  );

  const handleApplyFilter = useCallback(
    (selected: Record<string, string[] | null>) => {
      navigate({
        search: (prev) => ({
          ...prev,
          mime: selected.mime ? selected.mime[0] : "",
          page: 0,
        }),
      });
    },
    [navigate]
  );

  return (
    <FilesBrowser
      // Explicit key to reset the selected items when the path changes
      key={params.snapshotId}
      snapshot={snapshot}
      source={params.source}
      vfsEntry={vfsEntry}
      page={search.page}
      perPage={search.per_page}
      filters={{
        mime: selectedMimeFilter,
        filename: search.filename,
      }}
      onPageChange={onPageChange}
      onPathChange={onPathChange}
      onSnapshotChange={onSnapshotChange}
      preview={search.preview}
      onPreviewChange={onPreviewChange}
      view={search.view}
      onViewChange={onViewChange}
      actions={
        <Menu.Item
          leftIcon="not-equal"
          label="Compare with previous"
          href={{
            to: "/compare",
            search: { b: snapshot.identifier.Value, path: vfsEntry.path },
          }}
        />
      }
      breadcrumb={
        <Breadcrumbs>
          {breadcrumbsForSource(
            snapshot.identifier,
            params.source,
            snapshot.sources[params.source].importer.directory
          )}
        </Breadcrumbs>
      }
      fileBreadcrumb={
        <Breadcrumbs>
          {breadcrumbsForPath(
            snapshot.identifier,
            params.source,
            snapshot.sources[params.source].importer.directory,
            vfsEntry.path
          )}
        </Breadcrumbs>
      }
      onSearch={handleSearch}
      onRemoveFilter={handleRemoveFilter}
      onApplyFilter={handleApplyFilter}
      errorPageHref={{
        to: "/snapshots/$snapshotId/$source/errors",
        params: {
          snapshotId: params.snapshotId,
          source: params.source,
        },
        search: { path: vfsEntry.path },
      }}
    />
  );
}
