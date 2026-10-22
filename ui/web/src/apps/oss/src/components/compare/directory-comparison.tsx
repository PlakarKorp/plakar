import { Size } from "@plakar-ui/common/api/enrich";
import {
  EnrichedChildEntryType,
  SnapshotDirectoryQueryOptions,
  SnapshotType,
  useKlosetAPI,
} from "@plakar-ui/common/api/kloset";
import { Breadcrumbs, DiffModal, FileIcon } from "@plakar-ui/common/components";
import {
  diffDirectories,
  DirectoryDiffEntry,
  DirectoryDiffStatus,
} from "@plakar-ui/common/utils";
import {
  Alert,
  Checkbox,
  EmptyList,
  Icon,
  Skeleton,
  Tag,
  Typography,
} from "@plakar-ui/ui/components";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import clsx from "clsx";
import { useState } from "react";
import {
  Cell,
  Column,
  Row as TableRow,
  RowProps,
  Table,
  TableBody,
  TableHeader,
  TableLayout,
  Virtualizer,
} from "react-aria-components";

import { Delta } from "./delta";

// Past this many entries, a directory is compared on its first ones only.
const MAX_ENTRIES = 5000;

// Only the rows in view of the page are rendered.
const ROW_HEIGHT = 41;
const HEADING_HEIGHT = 40;

type Row = DirectoryDiffEntry<EnrichedChildEntryType>;

const STATUS: Record<
  DirectoryDiffStatus,
  { label: string; variant: "green" | "red" | "yellow" | "transparent" }
> = {
  added: { label: "Added", variant: "green" },
  removed: { label: "Removed", variant: "red" },
  changed: { label: "Changed", variant: "yellow" },
  unchanged: { label: "Unchanged", variant: "transparent" },
};

const ORDER: DirectoryDiffStatus[] = [
  "changed",
  "added",
  "removed",
  "unchanged",
];

type DirectoryComparisonProps = {
  before: SnapshotType;
  after: SnapshotType;
  // Where the two restore points overlap: the breadcrumb starts there.
  root: string;
  path: string;
};

/**
 * Compares a directory between two restore points, entry by entry: folders
 * open the same comparison one level down, changed files open their diff.
 */
export function DirectoryComparison({
  before,
  after,
  root,
  path,
}: DirectoryComparisonProps) {
  const { api, repository } = useKlosetAPI();
  const [showUnchanged, setShowUnchanged] = useState(false);
  const [diffing, setDiffing] = useState<Row | null>(null);

  const list = (snapshot: SnapshotType) =>
    SnapshotDirectoryQueryOptions(api, {
      repository,
      snapshot: snapshot.identifier.Value,
      source: 0,
      path,
      max: MAX_ENTRIES,
    });
  const a = useQuery({ ...list(before), placeholderData: keepPreviousData });
  const b = useQuery({ ...list(after), placeholderData: keepPreviousData });

  if (a.isPending || b.isPending) {
    return <DirectoryComparisonSkeleton root={root} path={path} />;
  }
  if (a.isError || b.isError) {
    return (
      <Alert variant="error" title="Unable to compare this folder">
        {(a.error ?? b.error)?.message}
      </Alert>
    );
  }
  if (!a.data && !b.data) {
    return (
      <Alert variant="default" title="Folder not found">
        {path} is in neither restore point.
      </Alert>
    );
  }

  // A truncated listing stops at its last name: past it, an entry of the
  // other side cannot be told apart from one that was cut off.
  const cutoff = [a.data, b.data]
    .filter((listing) => listing?.isTruncated)
    .map((listing) => listing?.children.at(-1)?.file_info.name ?? "")
    .sort()[0];
  const named = (children: EnrichedChildEntryType[] = []) =>
    children
      .map((child) => ({ ...child, name: child.file_info.name }))
      .filter((child) => cutoff === undefined || child.name <= cutoff);
  const rows = diffDirectories(
    named(a.data?.children),
    named(b.data?.children)
  );
  const counts: Record<DirectoryDiffStatus, number> = {
    added: 0,
    removed: 0,
    changed: 0,
    unchanged: 0,
  };
  for (const row of rows) counts[row.status]++;
  const visible = rows
    .filter((row) => showUnchanged || row.status !== "unchanged")
    .sort((x, y) => {
      const dx = isDir(x),
        dy = isDir(y);
      if (dx !== dy) return dx ? -1 : 1;
      return (
        ORDER.indexOf(x.status) - ORDER.indexOf(y.status) ||
        x.name.localeCompare(y.name)
      );
    });

  const target = (row: Row): Pick<RowProps<Row>, "href" | "onAction"> => {
    const entry = row.after ?? row.before;
    if (!entry) return {};
    if (isDir(row)) {
      return {
        href: {
          to: "/compare",
          search: (prev) => ({ ...prev, path: entry.path }),
        },
      };
    }
    if (row.status === "changed" && row.before && row.after) {
      return { onAction: () => setDiffing(row) };
    }
    // Other files open where they exist, in the browser.
    return { href: browseHref(row.after ? after : before, entry) };
  };

  return (
    <section
      className={clsx(
        "border-secondary bg-primary flex flex-col overflow-clip rounded-xl border transition-opacity",
        (a.isPlaceholderData || b.isPlaceholderData) && "opacity-60"
      )}
    >
      <PathBar root={root} path={path} />

      <div className="border-secondary bg-secondary flex flex-wrap items-center gap-2 border-b px-4 py-2">
        {ORDER.filter((s) => s !== "unchanged").map((status) => (
          <Tag key={status} variant={STATUS[status].variant} size="sm">
            {counts[status]} {STATUS[status].label.toLowerCase()}
          </Tag>
        ))}
        <span className="flex-1" />
        <Checkbox
          size="sm"
          isSelected={showUnchanged}
          onChange={setShowUnchanged}
          label={`Show ${counts.unchanged} unchanged`}
        />
      </div>

      {cutoff !== undefined && (
        <div className="p-4">
          <Alert variant="warning" title="Large folder">
            Only entries up to “{cutoff}” are compared.
          </Alert>
        </div>
      )}

      {visible.length === 0 ? (
        <EmptyList
          size="sm"
          title="No differences here"
          description="This folder is identical in both restore points."
        />
      ) : (
        <Virtualizer
          layout={TableLayout}
          layoutOptions={{
            estimatedRowHeight: ROW_HEIGHT,
            headingHeight: HEADING_HEIGHT,
          }}
        >
          <Table aria-label="Folder entries" className="-mb-px w-full">
            <TableHeader className="border-secondary bg-primary h-full border-b">
              <Column isRowHeader className="h-full" width="3fr" minWidth={200}>
                <HeaderLabel>Name</HeaderLabel>
              </Column>
              <Column className="h-full" width={120}>
                <HeaderLabel>Change</HeaderLabel>
              </Column>
              <Column className="h-full" width={110}>
                <HeaderLabel align="right">Before</HeaderLabel>
              </Column>
              <Column className="h-full" width={110}>
                <HeaderLabel align="right">After</HeaderLabel>
              </Column>
              <Column className="h-full" width={120}>
                <HeaderLabel align="right">Difference</HeaderLabel>
              </Column>
            </TableHeader>
            <TableBody>
              {visible.map((row) => (
                <ComparisonRow key={row.name} row={row} {...target(row)} />
              ))}
            </TableBody>
          </Table>
        </Virtualizer>
      )}

      {diffing?.before && diffing.after && (
        <DiffModal
          isOpen
          onOpenChange={(isOpen) => !isOpen && setDiffing(null)}
          source={0}
          before={{
            snapshot: before.identifier.Value,
            path: diffing.before.path,
            size: diffing.before.file_info.size.Value,
            label: before.identifier.toHumanString(),
            href: browseHref(before, diffing.before),
          }}
          after={{
            snapshot: after.identifier.Value,
            path: diffing.after.path,
            size: diffing.after.file_info.size.Value,
            label: after.identifier.toHumanString(),
            href: browseHref(after, diffing.after),
          }}
        />
      )}
    </section>
  );
}

function browseHref(
  snapshot: SnapshotType,
  entry: EnrichedChildEntryType
): RowProps<Row>["href"] {
  return {
    to: "/snapshots/$snapshotId/$source/browse",
    params: { snapshotId: snapshot.identifier.Value, source: 0 },
    search: { path: entry.parent_path, preview: entry.path },
  };
}

function isDir(row: Row) {
  return (row.after ?? row.before)?.file_info.mode.isDir() ?? false;
}

function sizeOf(entry?: EnrichedChildEntryType) {
  if (!entry) return undefined;
  return entry.file_info.mode.isDir()
    ? (entry.summary?.Total.size.Value ?? 0)
    : entry.file_info.size.Value;
}

function HeaderLabel({
  children,
  align = "left",
}: {
  children: string;
  align?: "left" | "right";
}) {
  return (
    <Typography
      size="xs"
      weight="medium"
      className={clsx(
        "text-tertiary flex h-full items-center px-4",
        align === "right" && "justify-end"
      )}
    >
      {children}
    </Typography>
  );
}

function ComparisonRow({
  row,
  href,
  onAction,
}: { row: Row } & Pick<RowProps<Row>, "href" | "onAction">) {
  const entry = row.after ?? row.before;
  const before = sizeOf(row.before);
  const after = sizeOf(row.after);
  const dir = isDir(row);
  return (
    <TableRow
      id={row.name}
      href={href}
      onAction={onAction}
      className={clsx(
        "border-secondary hover:bg-primary-hover data-focus-visible:outline-focus h-full cursor-pointer border-b outline-none data-focus-visible:outline-2 data-focus-visible:-outline-offset-2",
        row.status === "added" &&
          "shadow-[inset_3px_0_0_var(--border-color-success)]",
        row.status === "removed" &&
          "shadow-[inset_3px_0_0_var(--border-color-error)]",
        row.status === "changed" &&
          "shadow-[inset_3px_0_0_var(--border-color-warning)]"
      )}
    >
      <Cell className="flex items-center px-4 py-2">
        <div className="flex min-w-0 items-center gap-2">
          {dir ? (
            <Icon type="md-folder" size="lg" className="text-accent" />
          ) : (
            <FileIcon contentType={entry?.contentType} />
          )}
          <Typography as="span" size="sm" className="break-all">
            {row.name}
          </Typography>
          {row.status === "changed" && !dir && (
            <Typography as="span" size="xs" className="text-accent">
              view diff
            </Typography>
          )}
        </div>
      </Cell>
      <Cell className="flex h-full items-center px-4">
        <Tag variant={STATUS[row.status].variant} size="sm">
          {STATUS[row.status].label}
        </Tag>
      </Cell>
      <Cell className="flex h-full items-center justify-end px-4 whitespace-nowrap">
        <Typography as="span" size="sm" className="text-tertiary tabular-nums">
          {before !== undefined ? Size(before).Pretty : "—"}
        </Typography>
      </Cell>
      <Cell className="flex h-full items-center justify-end px-4 whitespace-nowrap">
        <Typography as="span" size="sm" className="tabular-nums">
          {after !== undefined ? Size(after).Pretty : "—"}
        </Typography>
      </Cell>
      <Cell className="flex h-full items-center justify-end px-4 whitespace-nowrap">
        <Delta
          before={before ?? 0}
          after={after ?? 0}
          format={(n) => Size(n).Pretty}
        />
      </Cell>
    </TableRow>
  );
}

/**
 * Holds the place of a folder comparison while its listings load.
 */
export function DirectoryComparisonSkeleton({
  root,
  path,
}: {
  root?: string;
  path?: string;
}) {
  return (
    <section className="border-secondary bg-primary flex flex-col rounded-xl border">
      {root && path ? (
        <PathBar root={root} path={path} />
      ) : (
        <div className="border-secondary border-b px-4 py-4">
          <Skeleton variant="text" size="md" width={280} />
        </div>
      )}
      <div className="border-secondary bg-secondary border-b px-4 py-2.5">
        <Skeleton variant="text" size="sm" width={240} />
      </div>
      <div className="flex flex-col gap-4 p-4">
        {Array.from({ length: 6 }, (_, i) => (
          <Skeleton key={i} variant="text" size="sm" width="100%" />
        ))}
      </div>
    </section>
  );
}

function PathBar({ root, path }: { root: string; path: string }) {
  // Breadcrumbs from where the restore points overlap down to this folder.
  const base = root.replace(/\/$/, "");
  const inRoot = path === root || path.startsWith(`${base}/`);
  const parts = (inRoot ? path.slice(base.length) : path)
    .split("/")
    .filter(Boolean);
  const start = inRoot ? root : "/";
  const crumbs = [
    { label: start, path: start },
    ...parts.map((part, i) => ({
      label: part,
      path: [inRoot ? base : "", ...parts.slice(0, i + 1)].join("/"),
    })),
  ];
  return (
    <div className="border-secondary flex items-center gap-2 border-b px-4 py-3">
      <Icon type="hard-drive" size="sm" className="text-tertiary" />
      <Breadcrumbs>
        {crumbs.map((crumb) => (
          <Breadcrumbs.Crumb
            key={crumb.path}
            to="/compare"
            search={(prev) => ({ ...prev, path: crumb.path })}
          >
            {crumb.label}
          </Breadcrumbs.Crumb>
        ))}
      </Breadcrumbs>
    </div>
  );
}
