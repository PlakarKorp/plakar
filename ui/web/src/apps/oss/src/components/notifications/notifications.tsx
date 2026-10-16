import { APIResponseBadRequest } from "@plakar-ui/common/api/errors";
import { addToast } from "@plakar-ui/common/components";
import {
  Button,
  EmptyList,
  LoginBanner,
  NotificationList,
  Spinner,
} from "@plakar-ui/ui/components";
import { useInfiniteQuery } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";

import { useSetNotificationStatus } from "~/api/account/mutations";
import { listAccountNotificationsQueryOptions } from "~/api/account/queries";

// Mapping between the notification level and the status to be displayed.
const STATUSES = {
  unknown: "default",
  info: "default",
  warning: "warning",
  error: "error",
  critical: "error",
} as const;

// Mapping between the notification level and the title to be displayed.
const TITLES = {
  unknown: undefined,
  info: undefined,
  warning: "Warning",
  error: "Error",
  critical: "Critical Error",
} as const;

type NotificationsListProps = {
  repository: string;
};

export function NotificationsList({ repository }: NotificationsListProps) {
  const query = useInfiniteQuery(
    listAccountNotificationsQueryOptions({ repository })
  );
  const navigate = useNavigate();

  const mutation = useSetNotificationStatus();

  if (query.isPending) {
    return <Spinner />;
  }

  if (query.isError) {
    return (
      <EmptyList
        illustrationType="sad"
        title="Something went wrong while fetching notifications."
        description={query.error.message}
      />
    );
  }

  const firstPage = query.data.pages[0];

  const allNotifications = query.data.pages
    .flatMap((page) => page.notifications?.items)
    .filter((v) => v !== undefined);

  if (firstPage.__type === "unauthenticated") {
    return <LoginBanner onClick={() => navigate({ to: "/login" })} />;
  }

  if (allNotifications.length === 0) {
    return <div className="text-sm">You have no unread notifications.</div>;
  }

  const hasUnreadNotification = allNotifications.some(
    (notification) => notification.status === "unread"
  );

  const readUnreadNotifications = () => {
    mutation.mutate(
      {
        notificationIds: allNotifications
          .filter((notification) => notification.status === "unread")
          .map((notification) => notification.id),
        status: "read",
      },
      {
        onError: (error) => {
          // On demo.plakar.io, the API returns a 400 error. Display the response payload as-is.
          if (error instanceof APIResponseBadRequest) {
            addToast({
              content: { children: error.responsePayload, variant: "error" },
            });
          } else {
            addToast({
              content: { children: error.toString(), variant: "error" },
            });
          }
        },
      }
    );
  };

  return (
    <div className="flex flex-col gap-3">
      {hasUnreadNotification && (
        <div className="self-end">
          <Button
            leftIcon="check"
            variant="secondary"
            size="sm"
            onPress={readUnreadNotifications}
          >
            Mark all as read
          </Button>
        </div>
      )}

      <NotificationList ariaLabel="Notifications">
        {allNotifications.map((notification) => (
          <NotificationList.Item
            key={notification.id}
            severity={STATUSES[notification.level]}
            title={TITLES[notification.level]}
            description={notification.message}
            date={notification.created_at.relativeToNow()}
            unread={notification.status === "unread"}
          />
        ))}
      </NotificationList>

      {query.hasNextPage && (
        <Button
          size="sm"
          variant="secondary"
          onPress={() => query.fetchNextPage()}
          isDisabled={query.isFetchingNextPage}
        >
          Load more
        </Button>
      )}
    </div>
  );
}
