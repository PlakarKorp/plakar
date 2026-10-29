import {
  Date as enrichDate,
  EnrichedDate,
  getIntegrationIcon,
} from "@plakar-ui/common/api/enrich";
import { Errors, Icon, Skeleton, Typography } from "@plakar-ui/ui/components";
import type { ToOptions } from "@tanstack/react-router";
import { Link } from "react-aria-components";

const DAY_FORMAT = "yyyy-MM-dd";

type RestorePointSource = {
  type: string;
  origin: string;
  directory: string;
};

export type RestorePointListItem = {
  id: string;
  createdAt: EnrichedDate;
  sources: RestorePointSource[];
  duration?: string;
  size: string;
  items: number;
  errors: number;
  link: ToOptions;
};

function dayLabel(date: EnrichedDate) {
  const day = date.format(DAY_FORMAT);
  const today = enrichDate(new Date().toISOString());
  const yesterday = enrichDate(new Date(Date.now() - 86_400_000).toISOString());
  if (day === today.format(DAY_FORMAT)) {
    return "Today";
  }
  if (day === yesterday.format(DAY_FORMAT)) {
    return "Yesterday";
  }
  return date.format("EEEE, MMM d, yyyy");
}

function itemsLabel(items: number) {
  return `${items.toLocaleString()} ${items === 1 ? "item" : "items"}`;
}

function sourcesLabel(sources: RestorePointSource[]) {
  const [first] = sources;
  const more = sources.length - 1;
  return more > 0
    ? `${first.directory} + ${more} more ${more === 1 ? "source" : "sources"}`
    : first.directory;
}

function RestorePointRow({ point }: { point: RestorePointListItem }) {
  return (
    <Link
      href={point.link}
      aria-label={`Restore point ${point.id}`}
      className="border-secondary hover:bg-primary-hover data-focus-visible:outline-focus flex items-center gap-4 border-b px-4 py-2.5 whitespace-nowrap outline-0 data-focus-visible:outline-2 data-focus-visible:-outline-offset-2"
    >
      <span className="text-secondary flex shrink-0 items-center gap-1">
        {point.sources.map((source, index) => (
          <span
            key={index}
            title={`${source.type} · ${source.origin}`}
            className="flex"
          >
            <Icon type={getIntegrationIcon(source.type)} size="lg" />
          </span>
        ))}
      </span>
      <Typography
        as="span"
        size="sm"
        weight="medium"
        className="text-primary w-11 shrink-0"
      >
        {point.createdAt.format("HH:mm")}
      </Typography>
      <Typography
        as="span"
        size="sm"
        className="text-tertiary hidden w-28 shrink-0 truncate md:block"
      >
        {point.createdAt.relativeToNow()}
      </Typography>
      <Typography
        as="span"
        size="sm"
        className="text-secondary min-w-0 flex-1 truncate"
      >
        {sourcesLabel(point.sources)}
      </Typography>
      <Errors length={point.errors} />
      <span className="ml-auto flex shrink-0 items-center gap-4">
        {point.duration && (
          <Typography
            as="span"
            size="sm"
            title="Duration"
            className="text-tertiary hidden w-14 items-center justify-end gap-1 tabular-nums md:flex"
          >
            <Icon type="clock" size="xs" />
            {point.duration}
          </Typography>
        )}
        <Typography
          as="span"
          size="sm"
          className="text-secondary w-16 text-right tabular-nums"
        >
          {point.size}
        </Typography>
        <Typography
          as="span"
          size="sm"
          className="text-tertiary hidden w-24 text-right tabular-nums md:block"
        >
          {itemsLabel(point.items)}
        </Typography>
      </span>
    </Link>
  );
}

export function RestorePointList({
  points,
}: {
  points: RestorePointListItem[];
}) {
  return (
    <div className="flex flex-col">
      {points.map((point, index) => {
        const label = dayLabel(point.createdAt);
        const isNewDay =
          index === 0 || dayLabel(points[index - 1].createdAt) !== label;

        return (
          <div key={point.id}>
            {isNewDay && (
              <Typography
                size="xs"
                weight="medium"
                className="bg-secondary border-secondary text-tertiary border-b px-4 py-1.5"
              >
                {label}
              </Typography>
            )}
            <RestorePointRow point={point} />
          </div>
        );
      })}
    </div>
  );
}

export function RestorePointListSkeleton({ rows }: { rows: number }) {
  return (
    <div className="flex flex-col">
      {Array.from({ length: rows }, (_, index) => (
        <div
          key={index}
          className="border-secondary flex items-center gap-4 border-b px-4 py-3"
        >
          <Skeleton variant="text" size="sm" className="w-full" />
        </div>
      ))}
    </div>
  );
}
