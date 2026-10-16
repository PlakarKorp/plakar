import clsx from "clsx";
import { PropsWithChildren } from "react";

import { Icon, IconType, Typography } from "..";
import { FeedItemStatuses } from "./feed.constants";
import { FeedSkeleton } from "./feed.skeleton";

type FeedProps = PropsWithChildren<{
  loading?: boolean;
}>;

function FeedRoot({ loading = false, children }: FeedProps) {
  if (loading) {
    return <FeedSkeleton />;
  }
  return <ol className="flex flex-col gap-4">{children}</ol>;
}

type FeedItemStatus = (typeof FeedItemStatuses)[number];

type FeedItemProps = {
  icon: IconType;
  status?: FeedItemStatus;
  title: string;
  description?: string;
  alt?: string;
};

function FeedItem({
  icon,
  status = "default",
  title,
  description,
  alt,
}: FeedItemProps) {
  return (
    <li className="group relative flex gap-3 py-2">
      <span
        aria-hidden="true"
        className="bg-border-secondary absolute top-10 left-4 -ml-px h-full w-0.5 group-last:hidden"
      />

      <div className="flex flex-1 gap-3">
        <div
          className={clsx(
            "flex h-8 w-8 shrink-0 items-center justify-center rounded-full",
            {
              default: "bg-tertiary text-primary",
              success: "bg-success-tertiary text-success",
              error: "bg-error-tertiary text-error",
              warning: "bg-warning-tertiary text-warning",
            }[status]
          )}
        >
          <Icon type={icon} size="md" />
        </div>

        <div className="flex w-full flex-col gap-1">
          <div className="flex justify-between">
            <Typography
              className="text-primary leading-none capitalize"
              size="md"
              weight="medium"
            >
              {title}
            </Typography>
            {alt !== undefined && (
              <Typography size="xs" className="text-secondary">
                {alt}
              </Typography>
            )}
          </div>

          {description !== undefined && (
            <Typography
              size="sm"
              className="text-secondary [overflow-wrap:anywhere]"
            >
              {description}
            </Typography>
          )}
        </div>
      </div>
    </li>
  );
}

export const Feed = Object.assign(FeedRoot, {
  Item: FeedItem,
});
