import { Size } from "@plakar-ui/common/api/enrich";
import {
  EnrichedChildEntryType,
  SnapshotDirectoryQueryOptions,
  SnapshotType,
  useKlosetAPI,
} from "@plakar-ui/common/api/kloset";
import { DiffModal, FileIcon } from "@plakar-ui/common/components";
import {
  diffDirectories,
  DirectoryDiffEntry,
  DirectoryDiffStatus,
} from "@plakar-ui/common/utils";
import {
  Alert,
  Checkbox,
  Icon,
  Spinner,
  Tag,
  Typography,
} from "@plakar-ui/ui/components";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import clsx from "clsx";
import { useState } from "react";
import { Button } from "react-aria-components";

// Past this many entries, a directory is compared on its first ones only.
const MAX_ENTRIES = 5000;

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
  onPathChange: (path: string) => void;
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
  onPathChange,
}: DirectoryComparisonProps) {
  const { api, repository } = useKlosetAPI();
  const navigate = useNavigate();
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
  const a = useQuery(list(before));
  const b = useQuery(list(after));

  if (a.isPending || b.isPending) {
    return <Spinner />;
  }
  if (a.isError || b.isError) {
    return (
      <Alert variant="error" title="Unable to compare this folder">
        {(a.error ?? b.error)?.message}
      </Alert>
    );
  }

  const named = (children: EnrichedChildEntryType[] = []) =>
    children.map((child) => ({ ...child, name: child.file_info.name }));
  const rows = diffDirectories(
    named(a.data?.children),
    named(b.data?.children)
  );
  const counts = Object.fromEntries(
    ORDER.map((status) => [
      status,
      rows.filter((r) => r.status === status).length,
    ])
  ) as Record<DirectoryDiffStatus, number>;
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

  const open = (row: Row) => {
    const entry = row.after ?? row.before;
    if (!entry) return;
    if (isDir(row)) {
      onPathChange(entry.path);
    } else if (row.status === "changed" && row.before && row.after) {
      setDiffing(row);
    } else {
      // Added or removed files open where they exist, in the browser.
      const snapshot = row.after ? after : before;
      void navigate({
        to: "/snapshots/$snapshotId/$source/browse",
        params: { snapshotId: snapshot.identifier.Value, source: 0 },
        search: { path: entry.parent_path, preview: entry.path },
      });
    }
  };

  return (
    <section className="border-secondary flex flex-col rounded-xl border">
      <PathBar root={root} path={path} onPathChange={onPathChange} />

      <div className="border-secondary flex flex-wrap items-center gap-2 border-b px-4 py-3">
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

      {!a.data && !b.data && (
        <div className="p-4">
          <Alert variant="default" title="Folder not found">
            {path} is in neither restore point.
          </Alert>
        </div>
      )}
      {(a.data?.isTruncated || b.data?.isTruncated) && (
        <div className="p-4">
          <Alert variant="warning" title="Large folder">
            Only its first {MAX_ENTRIES.toLocaleString()} entries are compared.
          </Alert>
        </div>
      )}

      {visible.length === 0 ? (
        <div className="flex flex-col items-center gap-2 p-10 text-center">
          <Icon type="success-circle" size="2xl" className="text-success" />
          <Typography weight="medium">No differences here</Typography>
          <Typography size="sm" className="text-tertiary">
            This folder is identical in both restore points.
          </Typography>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-secondary text-tertiary border-b text-left text-xs">
                <th className="px-4 py-2 font-medium">Name</th>
                <th className="px-4 py-2 font-medium">Change</th>
                <th className="px-4 py-2 text-right font-medium">Before</th>
                <th className="px-4 py-2 text-right font-medium">After</th>
                <th className="px-4 py-2 text-right font-medium">Difference</th>
              </tr>
            </thead>
            <tbody>
              {visible.map((row) => (
                <ComparisonRow
                  key={row.name}
                  row={row}
                  onOpen={() => open(row)}
                />
              ))}
            </tbody>
          </table>
        </div>
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
          }}
          after={{
            snapshot: after.identifier.Value,
            path: diffing.after.path,
            size: diffing.after.file_info.size.Value,
            label: after.identifier.toHumanString(),
          }}
        />
      )}
    </section>
  );
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

function ComparisonRow({ row, onOpen }: { row: Row; onOpen: () => void }) {
  const entry = row.after ?? row.before;
  const before = sizeOf(row.before);
  const after = sizeOf(row.after);
  const delta = (after ?? 0) - (before ?? 0);
  const dir = isDir(row);
  return (
    <tr
      className={clsx(
        "border-secondary hover:bg-secondary border-b last:border-0",
        row.status === "added" &&
          "shadow-[inset_3px_0_0_var(--border-color-success)]",
        row.status === "removed" &&
          "shadow-[inset_3px_0_0_var(--border-color-error)]",
        row.status === "changed" &&
          "shadow-[inset_3px_0_0_var(--border-color-warning)]"
      )}
    >
      <td className="px-4 py-2">
        <Button
          onPress={onOpen}
          className="data-focus-visible:outline-focus flex cursor-pointer items-center gap-2 text-left hover:underline data-focus-visible:outline-2"
        >
          {dir ? (
            <Icon type="md-folder" size="lg" className="text-accent" />
          ) : (
            <FileIcon contentType={entry?.object?.content_type} />
          )}
          <span className="break-all">{row.name}</span>
          {row.status === "changed" && !dir && (
            <Typography as="span" size="xs" className="text-tertiary">
              view diff
            </Typography>
          )}
        </Button>
      </td>
      <td className="px-4 py-2">
        <Tag variant={STATUS[row.status].variant} size="sm">
          {STATUS[row.status].label}
        </Tag>
      </td>
      <td className="text-tertiary px-4 py-2 text-right tabular-nums">
        {before !== undefined ? Size(before).Pretty : "—"}
      </td>
      <td className="px-4 py-2 text-right tabular-nums">
        {after !== undefined ? Size(after).Pretty : "—"}
      </td>
      <td
        className={clsx(
          "px-4 py-2 text-right tabular-nums",
          delta > 0
            ? "text-success"
            : delta < 0
              ? "text-error"
              : "text-tertiary"
        )}
      >
        {delta === 0
          ? "="
          : `${delta > 0 ? "+" : "−"}${Size(Math.abs(delta)).Pretty}`}
      </td>
    </tr>
  );
}

function PathBar({
  root,
  path,
  onPathChange,
}: {
  root: string;
  path: string;
  onPathChange: (path: string) => void;
}) {
  // Breadcrumbs from where the restore points overlap down to this folder.
  const relative = path.startsWith(root) ? path.slice(root.length) : path;
  const parts = relative.split("/").filter(Boolean);
  const crumbs = [
    { label: root, path: root },
    ...parts.map((part, i) => ({
      label: part,
      path: [root.replace(/\/$/, ""), ...parts.slice(0, i + 1)].join("/"),
    })),
  ];
  return (
    <nav
      aria-label="Folder"
      className="border-secondary flex flex-wrap items-center gap-1 border-b px-4 py-3 text-sm"
    >
      <Icon type="hard-drive" size="sm" className="text-tertiary" />
      {crumbs.map((crumb, i) => (
        <span key={crumb.path} className="flex items-center gap-1">
          {i > 0 && (
            <Icon type="chevron-right" size="xs" className="text-tertiary" />
          )}
          {i === crumbs.length - 1 ? (
            <span className="text-primary font-medium break-all">
              {crumb.label}
            </span>
          ) : (
            <Button
              onPress={() => onPathChange(crumb.path)}
              className="text-secondary cursor-pointer break-all hover:underline"
            >
              {crumb.label}
            </Button>
          )}
        </span>
      ))}
    </nav>
  );
}
