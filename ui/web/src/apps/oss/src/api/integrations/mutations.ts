import { useMutation, useQueryClient } from "@tanstack/react-query";

import { installIntegration, removeIntegration } from "./api";
import { integrationKeys } from "./keys";

import { getApiInfoQueryOptions } from "~/api/info/queries";

export function useInstallIntegration() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (...props: Parameters<typeof installIntegration>) => {
      const apiInfo = await queryClient.ensureQueryData(
        getApiInfoQueryOptions()
      );
      if (apiInfo.demo_mode) {
        throw new Error(
          "It is not possible to install integrations in demo mode."
        );
      }
      return installIntegration(...props);
    },
    onSettled: () => {
      queryClient.invalidateQueries({
        queryKey: integrationKeys.all(),
      });
    },
  });
}

export function useRemoveIntegration() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (...props: Parameters<typeof removeIntegration>) => {
      const apiInfo = await queryClient.ensureQueryData(
        getApiInfoQueryOptions()
      );
      if (apiInfo.demo_mode) {
        throw new Error(
          "It is not possible to remove integrations in demo mode."
        );
      }
      return removeIntegration(...props);
    },
    onSettled: () => {
      queryClient.invalidateQueries({
        queryKey: integrationKeys.all(),
      });
    },
  });
}
