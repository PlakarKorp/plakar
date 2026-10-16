import { IconButton } from "@plakar-ui/ui/components";
import { useInfiniteQuery } from "@tanstack/react-query";
import clsx from "clsx";

import { listAccountNotificationsQueryOptions } from "~/api/account/queries";

type NotificationsIconProps = {
  repository: string;
  onPress: () => void;
};

export function NotificationsIcon({
  repository,
  onPress,
}: NotificationsIconProps) {
  const query = useInfiniteQuery(
    listAccountNotificationsQueryOptions({ repository, status: "unread" })
  );
  const total =
    query.data?.pages[0].__type === "ok"
      ? query.data.pages[0].notifications.total
      : undefined;

  return (
    <span
      className={clsx(
        total !== undefined && total > 0 && "relative inline-block pr-2"
      )}
    >
      <IconButton
        aria-label="Open notifications"
        variant="tertiary"
        icon={total !== undefined && total > 0 ? "bell" : "bell"} // XXX: FA bell ring is not available in free version
        onPress={onPress}
        size="sm"
        tooltip="Notifications"
      />
      {total !== undefined && total > 0 && (
        <span className="pointer-events-none absolute -top-2 right-0 flex min-w-4 items-center justify-center rounded-full bg-red-500 px-0.75 text-xs font-medium text-white">
          {total > 99 ? "99+" : total}
        </span>
      )}
    </span>
  );
}
