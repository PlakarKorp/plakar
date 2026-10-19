import {
  Button,
  Icon,
  IconType,
  Menu,
  Skeleton,
  Typography,
} from "@plakar-ui/ui/components";
import { useInfiniteQuery } from "@tanstack/react-query";
import clsx from "clsx";
import { SyntheticEvent, useState } from "react";

import {
  RepositoryLocatePathname,
  RepositoryLocatePathnameQueryOptions,
  SnapshotType,
  useKlosetAPI,
  VFSEntryType,
} from "@plakar-ui/common/api/kloset";

import { DiffModal } from "./diff";

type TimelineProps = {
  snapshot: SnapshotType;
  source: number;
  vfsEntry: VFSEntryType;
  variant?: "default" | "compact";
  onSnapshotChange: (snapshotId: string) => void;
};

/**
 * Timeline component displays the activity of a path in other snapshots.
 *
 * The component displays:
 * - the list of snapshots containing the path that have already been fetched
 * - optionally, a "load more" button to fetch more snapshots.
 * - the last snapshot (ie. the oldest) containing the path.
 */
export function Timeline({
  snapshot,
  source,
  vfsEntry,
  variant = "default",
  onSnapshotChange,
}: TimelineProps) {
  const { api, repository } = useKlosetAPI();
  const [diffWith, setDiffWith] = useState<RepositoryLocatePathname | null>(
    null
  );
  const query = useInfiniteQuery(
    RepositoryLocatePathnameQueryOptions(api, snapshot, source, vfsEntry, {
      repository,
      pagination: {
        page: 0,
        perPage: 5,
      },
    })
  );

  if (query.error) {
    return (
      <>
        <Typography size="md" className="text-secondary">
          Unable to load restore point history
        </Typography>
        <Typography size="sm" className="text-secondary">
          An error occurred while fetching the restore point history. You can
          try to reload the page: {query.error.message}
        </Typography>
      </>
    );
  }

  if (query.isPending) {
    return (
      <Menu
        className={clsx(variant === "compact" && "max-h-[300px] min-w-[320px]")}
        unstyled={variant === "default"}
        aria-label="Loading timeline"
      >
        {Array.from({ length: 7 }).map((_, i) => (
          <Menu.Item
            key={i}
            leftIcon="clock"
            label={<Skeleton variant="text" size="sm" className="w-36" />}
            altText={
              <Skeleton variant="text" size="xs" className="w-[100px]" />
            }
            // @ts-expect-error: closeOnSelect isn't a valid prop for Menu.Item
            closeOnSelect={false}
            className={clsx(
              variant === "default" &&
                "border-primary mb-2 rounded-xl border p-4"
            )}
          />
        ))}
      </Menu>
    );
  }

  // Construct the list of entries to display in the feed: for each page, list
  // the snapshots.
  const entries = query.data.pages.map((page) => page.response.items).flat();

  const menuItems = entries.map((entry) =>
    getTimelineEntry(
      snapshot,
      entry,
      vfsEntry,
      onSnapshotChange,
      variant,
      setDiffWith
    )
  );

  // Total number of items.
  const total = query.data.pages[query.data.pages.length - 1].response.total;

  // Number of items that are not displayed.
  const moreToDisplay = total - menuItems.length;

  if (moreToDisplay > 0) {
    if (moreToDisplay > 1) {
      menuItems.push({
        id: "more",
        className: "border-0",
        label: (
          <Typography
            variant="body"
            size="sm"
            className={clsx(
              "text-tertiary",
              query.isFetching && "animate-pulse"
            )}
          >
            {moreToDisplay - 1} more restore point
            {moreToDisplay - 1 > 1 ? "s" : ""}
          </Typography>
        ),
        alt: (
          <Button
            variant="secondary"
            size={variant === "compact" ? "sm" : "md"}
            onPress={() => query.fetchNextPage()}
            isDisabled={query.isFetching}
            isPending={query.isFetching}
          >
            Load more
          </Button>
        ),
        icon: "circle-plus",
        closeOnSelect: false,
      });
    }
    // Never false because query.data.pages[0] is always set for the first page
    // if there is something to display, but TypeScript doesn't know that.
    if (query.data.pages[0].lastEntry) {
      menuItems.push(
        getTimelineEntry(
          snapshot,
          query.data.pages[0].lastEntry,
          vfsEntry,
          onSnapshotChange,
          variant,
          setDiffWith
        )
      );
    }
  }

  // The diff reads oldest to newest, whichever of the two is the current one.
  const diffSides =
    diffWith &&
    [
      { entry: diffWith, isCurrent: false },
      {
        entry: {
          snapshot,
          vfs_entry: vfsEntry,
        },
        isCurrent: true,
      },
    ].sort(
      (x, y) =>
        x.entry.snapshot.timestamp.Date.getTime() -
        y.entry.snapshot.timestamp.Date.getTime()
    );
  const toSide = (side: {
    entry: Pick<RepositoryLocatePathname, "snapshot" | "vfs_entry">;
    isCurrent: boolean;
  }) => ({
    snapshot: side.entry.snapshot.identifier.Value,
    path: side.entry.vfs_entry.path,
    size: side.entry.vfs_entry.file_info.size.Value,
    label: `${side.entry.snapshot.identifier.toHumanString()}${side.isCurrent ? " (this one)" : ""}`,
  });

  return (
    <>
      {diffSides && (
        <DiffModal
          isOpen
          onOpenChange={(open) => !open && setDiffWith(null)}
          source={source}
          before={toSide(diffSides[0])}
          after={toSide(diffSides[1])}
        />
      )}
      <Menu
        className={clsx(
          variant === "compact" && "max-h-[300px] min-w-[320px] overflow-auto"
        )}
        unstyled={variant === "default"}
        aria-label="Timeline"
      >
        {menuItems.map((item) => (
          <Menu.Item
            key={item.id}
            leftIcon={item.icon}
            label={item.label}
            altText={item.alt}
            onAction={item.onAction}
            className={clsx(
              variant === "default" && "mb-2 rounded-xl border p-4",
              item.className
            )}
            // @ts-expect-error: closeOnSelect isn't a valid prop for Menu.Item
            closeOnSelect={item.closeOnSelect}
          />
        ))}
      </Menu>
    </>
  );
}

// A plain button: the menu row around it acts on pointer and key events, so
// they must stop here for the press to open the diff instead of the version.
function DiffButton({ onPress }: { onPress: () => void }) {
  const stop = (e: SyntheticEvent) => e.stopPropagation();
  return (
    <button
      type="button"
      onPointerDown={stop}
      onPointerUp={stop}
      onMouseDown={stop}
      onMouseUp={stop}
      onKeyDown={(e) => {
        e.stopPropagation();
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          onPress();
        }
      }}
      onClick={(e) => {
        e.stopPropagation();
        onPress();
      }}
      className="border-primary bg-primary text-secondary hover:bg-primary-hover focus-visible:outline-focus flex cursor-pointer items-center gap-1 rounded-md border px-2 py-0.5 text-xs focus-visible:outline-2"
    >
      <Icon type="not-equal" size="xs" />
      View diff
    </button>
  );
}

type GetTimelineEntry = (
  snapshot: SnapshotType,
  entry: RepositoryLocatePathname,
  vfsEntry: VFSEntryType,
  onSnapshotChange: (snapshotId: string) => void,
  variant: "default" | "compact",
  onDiff: (entry: RepositoryLocatePathname) => void
) => {
  id: string;
  label: React.ReactNode;
  alt: React.ReactNode;
  icon: IconType;
  className?: string;
  closeOnSelect?: boolean;
  onAction?: () => void;
};

const getTimelineEntry: GetTimelineEntry = function (
  snapshot,
  entry,
  vfsEntry,
  onSnapshotChange,
  variant,
  onDiff
) {
  const isModified = vfsEntry.object
    ? vfsEntry.object.contentMAC !== entry.vfs_entry.object?.contentMAC
    : vfsEntry.mac !== entry.vfs_entry.mac;
  const isCurrent =
    snapshot.identifier.Value === entry.snapshot.identifier.Value;

  const handlePress = !isCurrent
    ? () => {
        onSnapshotChange(entry.snapshot.identifier.Value);
      }
    : undefined;

  return {
    id: entry.snapshot.identifier.Value,
    label: (
      <div className="group flex items-baseline gap-2">
        <Typography
          variant="body"
          size="sm"
          weight="medium"
          className={clsx("text-primary", isCurrent && "underline")}
          monospace
        >
          {entry.snapshot.identifier.toHumanString()}
        </Typography>

        {isModified && <Icon type="not-equal" size="xs" />}

        {!isCurrent && (
          <Typography
            variant="body"
            size={variant === "compact" ? "xs" : "sm"}
            className="text-tertiary"
          >
            {isModified ? "Differences" : "No changes"}
          </Typography>
        )}
      </div>
    ),
    alt:
      isModified && !isCurrent && vfsEntry.file_info.mode.isRegular() ? (
        <div className="flex items-center gap-2">
          <span>{entry.snapshot.timestamp.relativeToNow()}</span>
          <DiffButton onPress={() => onDiff(entry)} />
        </div>
      ) : (
        entry.snapshot.timestamp.relativeToNow()
      ),
    icon: "clock",
    className: isCurrent
      ? "border-accent-heavy"
      : "cursor-pointer border-primary",
    onAction: handlePress,
  };
};
