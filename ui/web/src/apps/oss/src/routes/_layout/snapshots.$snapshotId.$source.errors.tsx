import {
  GetSnapshotPathQueryOptions,
  GetSnapshotQueryOptions,
  SnapshotVFSErrorsQueryOptions,
  useKlosetAPI,
} from "@plakar-ui/common/api/kloset";
import {
  Breadcrumbs,
  ErrorsBrowser,
  MainTitle,
} from "@plakar-ui/common/components";
import { Main } from "@plakar-ui/common/layouts";
import { createPaginationSearchSchema } from "@plakar-ui/common/utils";
import { useQuery, useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute, redirect } from "@tanstack/react-router";
import std_path from "path";
import { z } from "zod";

import {
  breadcrumbsForPath,
  breadcrumbsForSource,
} from "~/utils/breadcrumbs-for-path";

export const Route = createFileRoute(
  "/_layout/snapshots/$snapshotId/$source/errors"
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

    context.queryClient.prefetchQuery(
      GetSnapshotQueryOptions(context.klosetAPI, {
        repository: undefined,
        snapshot: params.snapshotId,
      })
    );

    const getSnapshotPathResponse = await context.queryClient.ensureQueryData(
      GetSnapshotPathQueryOptions(context.klosetAPI, {
        repository: undefined,
        snapshot: params.snapshotId,
        source: params.source,
        path: deps.path,
      })
    );

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
  },

  component: Component,
});

function Component() {
  const { snapshotId, source } = Route.useParams();
  const { path, page, per_page: perPage } = Route.useSearch();
  const navigate = Route.useNavigate();
  const { api } = useKlosetAPI();
  const query = useQuery({
    ...SnapshotVFSErrorsQueryOptions(api, {
      repository: undefined,
      snapshot: snapshotId,
      source,
      path,
      pagination: {
        page,
        perPage,
      },
    }),
  });
  const snapshotQuery = useSuspenseQuery(
    GetSnapshotQueryOptions(api, {
      repository: undefined,
      snapshot: snapshotId,
    })
  );

  const onPageChange = (page: number, per_page: number) =>
    navigate({
      search: (prev) => ({
        ...prev,
        page: prev.per_page !== per_page ? 0 : page,
        per_page,
      }),
    });

  return (
    <Main
      title={<MainTitle snapshot={snapshotQuery.data.item} source={source} />}
      breadcrumb={
        <Breadcrumbs>
          {breadcrumbsForSource(
            snapshotQuery.data.item.identifier,
            source,
            snapshotQuery.data.item.sources[source].importer.directory
          )}
        </Breadcrumbs>
      }
    >
      <div className="flex flex-col gap-4">
        <Breadcrumbs>
          {breadcrumbsForPath(
            snapshotQuery.data.item.identifier,
            source,
            snapshotQuery.data.item.sources[source].importer.directory,
            path
          )}
          <Breadcrumbs.Crumb
            key="currentPath"
            to="/snapshots/$snapshotId/$source/errors"
            params={{ snapshotId, source }}
          >
            Errors
          </Breadcrumbs.Crumb>
        </Breadcrumbs>

        <ErrorsBrowser
          query={query}
          page={page}
          perPage={perPage}
          onPageChange={onPageChange}
        />
      </div>
    </Main>
  );
}
