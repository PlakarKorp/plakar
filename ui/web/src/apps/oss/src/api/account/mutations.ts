import { useMutation, useQueryClient } from "@tanstack/react-query";

import {
  getAccountServicesAlerting,
  setAccountServicesAlerting,
  setNotificationStatus,
} from "./api";
import { accountKeys } from "./keys";

import { getApiInfoQueryOptions } from "~/api/info/queries";

export function useSetNotificationStatus() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (...props: Parameters<typeof setNotificationStatus>) => {
      const apiInfo = await queryClient.ensureQueryData(
        getApiInfoQueryOptions()
      );
      if (apiInfo.demo_mode) {
        throw new Error(
          "It is not possible to mark notifications as read in demo mode."
        );
      }
      return setNotificationStatus(...props);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: accountKeys.notifications(),
      });
    },
  });
}

export function useAccountServicesSetAlerting() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (
      ...props: Parameters<typeof setAccountServicesAlerting>
    ) => {
      const apiInfo = await queryClient.ensureQueryData(
        getApiInfoQueryOptions()
      );
      if (apiInfo.demo_mode) {
        throw new Error(
          "It is not possible to change the alerting configuration in demo mode."
        );
      }
      return setAccountServicesAlerting(...props);
    },
    onMutate: ({ enabled, email_report }) => {
      queryClient.setQueryData<
        Awaited<ReturnType<typeof getAccountServicesAlerting>>
      >(accountKeys.servicesAlerting(), {
        __type: "ok",
        configuration: { enabled, email_report },
      });
    },
    onError: () => {
      queryClient.invalidateQueries({
        queryKey: accountKeys.servicesAlerting(),
      });
    },
  });
}
