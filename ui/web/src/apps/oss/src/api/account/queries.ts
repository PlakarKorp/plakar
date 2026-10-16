import { infiniteQueryOptions, queryOptions } from "@tanstack/react-query";

import {
  getAccountMe,
  getAccountNotifications,
  getAccountServicesAlerting,
} from "./api";
import { accountKeys } from "./keys";

export const getAccountMeQueryOptions = () =>
  queryOptions({
    queryKey: accountKeys.me(),
    queryFn: getAccountMe,
  });

export const listAccountNotificationsQueryOptions = ({
  repository,
  status,
}: {
  repository: string;
  status?: Parameters<typeof getAccountNotifications>[0]["status"];
}) =>
  infiniteQueryOptions({
    queryKey: accountKeys.notificationsList(repository, status),
    refetchInterval: 60 * 1000,
    queryFn: async ({ pageParam }) => {
      const notifications = await getAccountNotifications({
        repository,
        pagination: {
          page: pageParam.page,
          perPage: pageParam.perPage,
        },
        status,
      });
      return {
        ...notifications,
        page: pageParam.page,
        perPage: pageParam.perPage,
      };
    },
    initialPageParam: { page: 0, perPage: 50 },
    getNextPageParam: (lastPage) => {
      if (lastPage.__type === "unauthenticated") {
        return null;
      }
      if (
        (lastPage.page + 1) * lastPage.perPage >=
        lastPage.notifications.total
      ) {
        return null;
      }
      return { page: lastPage.page + 1, perPage: lastPage.perPage };
    },
  });

export const getAccountServicesAlertingQueryOptions = () =>
  queryOptions({
    queryKey: accountKeys.servicesAlerting(),
    queryFn: getAccountServicesAlerting,
  });
