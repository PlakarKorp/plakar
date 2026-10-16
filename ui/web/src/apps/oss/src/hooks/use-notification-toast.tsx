import { APIResponseBadRequest } from "@plakar-ui/common/api/errors";
import { addToast } from "@plakar-ui/common/components";
import { useInfiniteQuery } from "@tanstack/react-query";
import { useEffect, useRef } from "react";

import { useSetNotificationStatus } from "~/api/account/mutations";
import { listAccountNotificationsQueryOptions } from "~/api/account/queries";

export function useNotificationToast(repository: string) {
  const hasFiredRef = useRef(false);

  const { data, isPending } = useInfiniteQuery(
    listAccountNotificationsQueryOptions({ repository, status: "unread" })
  );

  const { mutate } = useSetNotificationStatus();

  const notifications = data?.pages
    .map((page) => page.notifications?.items)
    .flat()
    .filter((notification) => notification?.flags.isBanner)
    .reverse();

  const banner = notifications?.[0];
  const bannerLevel = banner?.level ?? "unknown";
  const bannerId = banner?.id;
  const bannerMessage = banner?.message;
  const isBanner = banner?.flags.isBanner;

  const alertLevel = (
    {
      unknown: "default",
      info: "default",
      warning: "warning",
      error: "error",
      critical: "error",
    } as const
  )[bannerLevel];

  useEffect(() => {
    if (hasFiredRef.current || isPending || !bannerId || !isBanner) {
      return;
    }

    addToast({
      content: {
        variant: alertLevel,
        title: bannerLevel,
        buttonProps: {
          children: "Mark as read",
          variant: "secondary",
          slot: "close",
          onPress: () => {
            mutate(
              {
                notificationIds: [bannerId],
                status: "read",
              },
              {
                onError: (error) => {
                  // On demo.plakar.io, the API returns a 400 error. Display the response payload as-is.
                  if (error instanceof APIResponseBadRequest) {
                    addToast({
                      content: {
                        children: error.responsePayload,
                        variant: "error",
                      },
                    });
                  } else {
                    addToast({
                      content: { children: error.toString(), variant: "error" },
                    });
                  }
                },
              }
            );
          },
        },
        children: bannerMessage,
        closable: false,
      },
      sticky: true,
    });

    hasFiredRef.current = true;
  }, [
    isPending,
    alertLevel,
    bannerId,
    bannerLevel,
    bannerMessage,
    isBanner,
    mutate,
  ]);
}
