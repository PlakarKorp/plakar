import clsx from "clsx";
import { PropsWithChildren } from "react";
import { GridList, GridListItem } from "react-aria-components";

import { Typography } from "..";
import { NotificationSeverity } from "./notification-list.constants";

type NotificationRootProps = PropsWithChildren<{
  ariaLabel: string;
}>;

function NotificationListRoot({ ariaLabel, children }: NotificationRootProps) {
  return (
    <GridList className="flex flex-col gap-4" aria-label={ariaLabel}>
      {children}
    </GridList>
  );
}

const SEVERITY_CLASSES: Record<NotificationSeverity, string> = {
  default: "border-accent text-primary",
  warning: "border-warning text-warning",
  error: "border-error text-error",
};

const UNREAD_BADGE_CLASSES: Record<NotificationSeverity, string> = {
  default: "bg-accent",
  warning: "bg-warning",
  error: "bg-error",
};

type NotificationItemProps = {
  severity: NotificationSeverity;
  title?: string;
  description: string;
  date: string;
  unread?: boolean;
};

function NotificationListItem({
  severity = "default",
  title,
  description,
  date,
  unread = false,
}: NotificationItemProps) {
  return (
    <GridListItem
      textValue={description}
      className={clsx(
        "bg-secondary flex justify-between rounded-l-sm rounded-r-lg border-l-4 p-3 break-words",
        SEVERITY_CLASSES[severity]
      )}
    >
      <div className="flex flex-col gap-3">
        <div>
          {title && (
            <Typography size="md" weight="medium">
              {title}
            </Typography>
          )}
          <Typography className="text-secondary" size="xs">
            {date}
          </Typography>
        </div>

        <Typography className="text-primary" size="sm">
          {description}
        </Typography>
      </div>

      {unread && (
        <div
          className={clsx(
            "h-[10px] w-[10px] shrink-0 animate-pulse rounded-full",
            UNREAD_BADGE_CLASSES[severity]
          )}
        />
      )}
    </GridListItem>
  );
}

export const NotificationList = Object.assign(NotificationListRoot, {
  Item: NotificationListItem,
});
