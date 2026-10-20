import {
  GetSnapshotQueryOptions,
  RepositorySnapshotsQueryOptions,
  SnapshotType,
  useSnapshotsAsyncList,
} from "@plakar-ui/common/api/kloset";
import { Main } from "@plakar-ui/common/layouts";
import {
  Alert,
  AsyncSelect,
  IconButton,
  SelectOption,
  Skeleton,
} from "@plakar-ui/ui/components";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { ReactNode } from "react";
import { Key } from "react-aria-components";
import { z } from "zod";

import { CompareSummary } from "~/components/compare/compare-summary";
import {
  DirectoryComparison,
  DirectoryComparisonSkeleton,
} from "~/components/compare/directory-comparison";

// Restore points per page, at the API maximum: the first page also holds the
// default comparison, the pickers load the next ones as they are scrolled.
const PAGE_SIZE = 100;

export const Route = createFileRoute("/_layout/compare")({
  validateSearch: z.object({
    // The older restore point and the newer one, by ID, or by ID prefix
    // among the listed ones. Left out, the newest one is compared with the
    // one before it of its source.
    a: z.string().optional().catch(undefined),
    b: z.string().optional().catch(undefined),
    path: z.string().optional().catch(undefined),
  }),
  component: ComparePage,
});

function sameSource(x: SnapshotType, y: SnapshotType) {
  const sx = x.sources[0]?.importer;
  const sy = y.sources[0]?.importer;
  return (
    sx?.type === sy?.type &&
    sx?.origin === sy?.origin &&
    sx?.directory.Value === sy?.directory.Value
  );
}

/**
 * Returns the deepest directory both paths are in, so that comparing two
 * sources starts where they overlap.
 */
function commonPath(x: string, y: string) {
  const a = x.split("/");
  const b = y.split("/");
  const out: string[] = [];
  for (let i = 0; i < Math.min(a.length, b.length) && a[i] === b[i]; i++) {
    out.push(a[i]);
  }
  return out.join("/") || "/";
}

/**
 * Resolves a restore point given in the URL: from the listed ones when the
 * ID or prefix matches one, else by asking for that exact ID.
 */
function useRestorePoint(listed: SnapshotType[] | undefined, id?: string) {
  const ctx = Route.useRouteContext();
  const match = id
    ? listed?.find((s) => s.identifier.Value.startsWith(id))
    : undefined;
  const query = useQuery({
    ...GetSnapshotQueryOptions(ctx.klosetAPI, {
      repository: undefined,
      snapshot: id ?? "",
    }),
    enabled: !!id && !!listed && !match,
  });
  return {
    snapshot: match ?? query.data?.item,
    isPending: !!id && !match && query.isPending,
    isError: !!id && !match && query.isError,
  };
}

function toOption(snapshot: SnapshotType): SelectOption {
  return {
    id: snapshot.identifier.Value,
    name: `${snapshot.identifier.toHumanString()} · ${snapshot.sources[0]?.importer.directory.toHumanString() ?? ""} · ${snapshot.timestamp.relativeToNow()}`,
  };
}

function ComparePage() {
  const search = Route.useSearch();
  const navigate = Route.useNavigate();
  const ctx = Route.useRouteContext();
  const query = useQuery(
    RepositorySnapshotsQueryOptions(ctx.klosetAPI, {
      repository: undefined,
      pagination: { page: 0, perPage: PAGE_SIZE },
    })
  );
  const requestedA = useRestorePoint(query.data?.items, search.a);
  const requestedB = useRestorePoint(query.data?.items, search.b);
  const snapshotsList = useSnapshotsAsyncList(PAGE_SIZE);
  const list = { ...snapshotsList, items: snapshotsList.items.map(toOption) };

  const isLoading =
    query.isPending || requestedA.isPending || requestedB.isPending;

  if (query.isError) {
    return (
      <ComparePageLayout>
        <Alert variant="error" title="Unable to list the restore points">
          {query.error.message}
        </Alert>
      </ComparePageLayout>
    );
  }
  const missing = [
    requestedA.isError ? search.a : undefined,
    requestedB.isError ? search.b : undefined,
  ].filter((id): id is string => !!id);
  if (missing.length > 0) {
    return (
      <ComparePageLayout>
        <Alert variant="error" title="Restore point not found">
          No restore point matches {missing.join(" or ")}.
        </Alert>
      </ComparePageLayout>
    );
  }

  const snapshots = query.data?.items ?? [];
  if (!isLoading && snapshots.length < 2) {
    return (
      <ComparePageLayout>
        <Alert variant="default" title="Nothing to compare yet">
          Comparing needs at least two restore points.
        </Alert>
      </ComparePageLayout>
    );
  }

  const after = isLoading ? undefined : (requestedB.snapshot ?? snapshots[0]);
  const before =
    requestedA.snapshot ??
    (after &&
      snapshots.find(
        (s) =>
          s.identifier.Value !== after.identifier.Value &&
          sameSource(s, after) &&
          s.timestamp.Date < after.timestamp.Date
      ));

  const pick = (side: "a" | "b") => (key: Key | Key[] | null) => {
    if (key === null || Array.isArray(key)) return;
    navigate({
      search: (prev) => ({ ...prev, [side]: String(key), path: undefined }),
    });
  };

  const isSame = before?.identifier.Value === after?.identifier.Value;
  const root = commonPath(
    before?.sources[0]?.importer.directory.Value ?? "/",
    after?.sources[0]?.importer.directory.Value ?? "/"
  );

  return (
    <ComparePageLayout>
      <div className="flex flex-wrap items-end gap-3">
        <AsyncSelect
          label="Before"
          placeholder="Pick a restore point"
          className="min-w-0 flex-1 basis-80"
          list={list}
          search={false}
          selectedOption={before && toOption(before)}
          value={before?.identifier.Value ?? null}
          onChange={pick("a")}
        />
        <IconButton
          icon="right-left"
          aria-label="Swap the restore points"
          tooltip="Swap"
          variant="secondary"
          isDisabled={!before || !after}
          onPress={() =>
            navigate({
              search: (prev) => ({
                ...prev,
                a: after?.identifier.Value,
                b: before?.identifier.Value,
              }),
            })
          }
        />
        <AsyncSelect
          label="After"
          className="min-w-0 flex-1 basis-80"
          list={list}
          search={false}
          selectedOption={after && toOption(after)}
          value={after?.identifier.Value ?? null}
          onChange={pick("b")}
        />
      </div>

      {isLoading ? (
        <>
          <Skeleton
            variant="rectangle"
            width="100%"
            height={222}
            className="rounded-xl"
          />
          <DirectoryComparisonSkeleton />
        </>
      ) : !before || !after ? (
        <Alert variant="default" title="No earlier restore point">
          None of the {PAGE_SIZE} most recent restore points is an earlier one
          of this source. Pick the one to compare with.
        </Alert>
      ) : (
        <>
          <CompareSummary before={before} after={after} />

          {isSame ? (
            <Alert variant="default" title="Same restore point on both sides">
              Pick two different restore points.
            </Alert>
          ) : (
            <DirectoryComparison
              before={before}
              after={after}
              root={root}
              path={search.path ?? root}
            />
          )}
        </>
      )}
    </ComparePageLayout>
  );
}

function ComparePageLayout({ children }: { children: ReactNode }) {
  return (
    <Main
      title="Compare restore points"
      description="What changed between two restore points, folder by folder."
    >
      <div className="flex flex-col gap-6">{children}</div>
    </Main>
  );
}
