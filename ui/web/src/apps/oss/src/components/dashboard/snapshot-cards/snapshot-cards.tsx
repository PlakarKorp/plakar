import {
  Errors,
  ExpandableList,
  IconButton,
  Id,
  Skeleton,
  Typography,
} from "@plakar-ui/ui/components";
import type { ToOptions } from "@tanstack/react-router";
import clsx from "clsx";
import { PropsWithChildren } from "react";
import {
  Cell,
  Column,
  Row,
  Table,
  TableBody,
  TableHeader,
} from "react-aria-components";

type SnapshotTableProps = PropsWithChildren<{
  className?: string;
}>;

function SnapshotsTable({ children, className }: SnapshotTableProps) {
  return (
    <Table
      aria-label="Restore points"
      className={clsx(
        "text-primary w-full table-auto border-spacing-x-4",
        className
      )}
    >
      <TableHeader className="sr-only">
        <Column isRowHeader>ID</Column>
        <Column>Source</Column>
        <Column>Date modified</Column>
        <Column>Duration</Column>
        <Column>Size</Column>
        <Column />
      </TableHeader>
      <TableBody>{children}</TableBody>
    </Table>
  );
}

type Source = {
  type?: string;
  origin?: string;
  directory?: string;
};

type SnapshotRowProps = {
  id: string;
  dateModified?: string;
  timeSince?: string;
  duration?: string;
  sources?: Source[];
  size?: string;
  itemsLength?: number;
  errorsLength?: number;
  snapshotLink?: ToOptions;
};

function SnapshotsRow({
  id,
  dateModified,
  timeSince,
  duration,
  sources,
  size,
  itemsLength,
  errorsLength,
  snapshotLink,
}: SnapshotRowProps) {
  const itemLabel =
    itemsLength !== undefined
      ? `${itemsLength} ${itemsLength > 1 ? "items" : "item"}`
      : undefined;

  return (
    <Row
      className={clsx(
        "border-secondary group mx-4 border-b last:border-0",
        snapshotLink && "cursor-pointer"
      )}
      href={snapshotLink}
    >
      <Cell className="py-4 pr-2 align-top">
        <div className="flex flex-col gap-1">
          <Id value={id} className="text-secondary" />
          <Errors length={errorsLength} />
        </div>
      </Cell>

      <Cell
        className={clsx("align-top", sources && "w-[40%] max-w-0 px-2 py-4")}
      >
        {sources && (
          <ExpandableList
            items={sources.map((source) => ({
              tag: source.type,
              value: source.origin ?? "",
              subValue: source.directory,
            }))}
          />
        )}
      </Cell>

      <Cell className={clsx("align-top", timeSince && "px-2 py-4")}>
        <LabeledText value={timeSince} subValue={dateModified} />
      </Cell>

      <Cell className={clsx("align-top", duration && "px-2 py-4")}>
        <LabeledText
          value={duration}
          subValue={duration ? "Duration" : undefined}
        />
      </Cell>

      <Cell className={clsx("align-top", size && "px-2 py-4")}>
        <LabeledText value={size} subValue={itemLabel} />
      </Cell>

      <Cell className={clsx("w-[36px] align-top", snapshotLink && "px-2 py-4")}>
        {snapshotLink && (
          <div className="h-[36px] w-[36px]">
            <div className="invisible group-hover:visible max-md:visible">
              <IconButton
                aria-label="View restore point"
                icon="arrow-right"
                variant="tertiary"
                href={snapshotLink}
                tooltip="View restore point"
              />
            </div>
          </div>
        )}
      </Cell>
    </Row>
  );
}

function LabeledText({
  value,
  subValue,
}: {
  value?: string;
  subValue?: string;
}) {
  if (!value && !subValue) return null;

  return (
    <div className="flex flex-col gap-1">
      {value && (
        <Typography
          variant="body"
          size="md"
          weight="medium"
          className="md:overflow-hidden md:text-ellipsis md:whitespace-nowrap"
        >
          {value}
        </Typography>
      )}
      {subValue && (
        <Typography
          variant="body"
          size="sm"
          className="text-secondary md:overflow-hidden md:text-ellipsis md:whitespace-nowrap"
        >
          {subValue}
        </Typography>
      )}
    </div>
  );
}

type SnapshotsRowSkeletonProps = {
  withSources?: boolean;
  withTimeSince?: boolean;
  withDuration?: boolean;
  withSize?: boolean;
  withSnapshotLink?: boolean;
};

function SnapshotsRowSkeleton({
  withSources = true,
  withTimeSince = true,
  withDuration = true,
  withSize = true,
  withSnapshotLink = true,
}: SnapshotsRowSkeletonProps) {
  return (
    <Row className="border-secondary mx-4 border-b last:border-0">
      <Cell className="py-4 pr-2 align-top">
        <Skeleton variant="text" size="md" />
      </Cell>

      <Cell className={clsx(withSources && "w-[40%] max-w-0 px-2 py-4")}>
        {withSources && (
          <div className="flex flex-col gap-1">
            <Skeleton variant="text" size="md" />
            <Skeleton variant="text" size="sm" />
          </div>
        )}
      </Cell>

      <Cell className={clsx(withTimeSince && "px-2 py-4")}>
        {withTimeSince && (
          <div className="flex flex-col gap-1">
            <Skeleton variant="text" size="md" />
            <Skeleton variant="text" size="sm" />
          </div>
        )}
      </Cell>

      <Cell className={clsx(withDuration && "px-2 py-4")}>
        {withDuration && (
          <div className="flex flex-col gap-1">
            <Skeleton variant="text" size="md" />
            <Skeleton variant="text" size="sm" />
          </div>
        )}
      </Cell>

      <Cell className={clsx(withSize && "px-2 py-4")}>
        {withSize && (
          <div className="flex flex-col gap-1">
            <Skeleton variant="text" size="md" />
            <Skeleton variant="text" size="sm" />
          </div>
        )}
      </Cell>

      <Cell className={clsx(withSnapshotLink && "w-[36px] px-2 py-4")} />
    </Row>
  );
}

export const Snapshots = Object.assign(SnapshotsTable, {
  Row: SnapshotsRow,
  RowSkeleton: SnapshotsRowSkeleton,
});
