import {
  RepositorySnapshotsQueryOptions,
  SnapshotType,
} from "@plakar-ui/common/api/kloset";
import { Main } from "@plakar-ui/common/layouts";
import { Alert, IconButton, Select, Spinner } from "@plakar-ui/ui/components";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { CompareSummary } from "~/components/compare/compare-summary";
import { DirectoryComparison } from "~/components/compare/directory-comparison";

// Restore points offered in the pickers, most recent first: one page, at
// the API maximum.
const PICKER_LIMIT = 100;

export const Route = createFileRoute("/_layout/compare")({
  validateSearch: z.object({
    // The older restore point and the newer one, by ID or ID prefix. Left
    // out, the newest one is compared with the one before it of its source.
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

function ComparePage() {
  const search = Route.useSearch();
  const navigate = Route.useNavigate();
  const ctx = Route.useRouteContext();
  const query = useQuery(
    RepositorySnapshotsQueryOptions(ctx.klosetAPI, {
      repository: undefined,
      pagination: { page: 0, perPage: PICKER_LIMIT },
    })
  );

  if (query.isPending) {
    return (
      <Main title="Compare restore points">
        <Spinner />
      </Main>
    );
  }
  if (query.isError) {
    return (
      <Main title="Compare restore points">
        <Alert variant="error" title="Unable to list the restore points">
          {query.error.message}
        </Alert>
      </Main>
    );
  }

  const snapshots = query.data.items;
  const find = (prefix?: string) =>
    prefix
      ? snapshots.find((s) => s.identifier.Value.startsWith(prefix))
      : undefined;
  const after = find(search.b) ?? snapshots[0];
  const before =
    find(search.a) ??
    snapshots.find(
      (s) =>
        s !== after &&
        sameSource(s, after) &&
        s.timestamp.Date < after.timestamp.Date
    ) ??
    snapshots.find((s) => s !== after);

  if (!after || !before) {
    return (
      <Main title="Compare restore points">
        <Alert variant="default" title="Nothing to compare yet">
          Comparing needs at least two restore points.
        </Alert>
      </Main>
    );
  }

  const root = commonPath(
    before.sources[0]?.importer.directory.Value ?? "/",
    after.sources[0]?.importer.directory.Value ?? "/"
  );
  const path = search.path ?? root;

  const options = snapshots.map((s) => ({
    id: s.identifier.Value,
    name: `${s.identifier.toHumanString()} · ${s.sources[0]?.importer.directory.toHumanString() ?? ""} · ${s.timestamp.relativeToNow()}`,
  }));

  const pick = (side: "a" | "b") => (key: unknown) => {
    if (typeof key !== "string") return;
    navigate({
      search: (prev) => ({ ...prev, [side]: key, path: undefined }),
    });
  };

  return (
    <Main
      title="Compare restore points"
      description="What changed between two restore points, folder by folder."
    >
      <div className="flex flex-col gap-6">
        <div className="flex flex-wrap items-end gap-3">
          <Select
            label="Before"
            className="min-w-0 flex-1 basis-80"
            value={before.identifier.Value}
            onChange={pick("a")}
          >
            <Select.ListBox>
              {options.map((option) => (
                <Select.Item key={option.id} {...option} />
              ))}
            </Select.ListBox>
          </Select>
          <IconButton
            icon="arrows-rotate"
            aria-label="Swap the restore points"
            tooltip="Swap"
            variant="secondary"
            onPress={() =>
              navigate({
                search: (prev) => ({
                  ...prev,
                  a: after.identifier.Value,
                  b: before.identifier.Value,
                }),
              })
            }
          />
          <Select
            label="After"
            className="min-w-0 flex-1 basis-80"
            value={after.identifier.Value}
            onChange={pick("b")}
          >
            <Select.ListBox>
              {options.map((option) => (
                <Select.Item key={option.id} {...option} />
              ))}
            </Select.ListBox>
          </Select>
        </div>

        <CompareSummary before={before} after={after} />

        {before === after ? (
          <Alert variant="default" title="Same restore point on both sides">
            Pick two different restore points.
          </Alert>
        ) : (
          <DirectoryComparison
            before={before}
            after={after}
            root={root}
            path={path}
            onPathChange={(next) =>
              navigate({ search: (prev) => ({ ...prev, path: next }) })
            }
          />
        )}
      </div>
    </Main>
  );
}
