import { useMutation, useQueryClient } from "@tanstack/react-query";

import { KlosetAPI } from "./api";
import { klosetKeys } from "./keys";
import { createMountFn } from "./schema";

type useCreateMountParams = {
  api: KlosetAPI;
  repository: string | undefined;
};

export const useCreateMount = ({ api, repository }: useCreateMountParams) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (props: Omit<Parameters<createMountFn>[0], "repository">) => {
      if (!api.mount) {
        throw new Error("Mount API is not available");
      }
      return api.mount.create({
        repository,
        ...props,
      });
    },
    onSuccess: () => {
      // XXX: better invalidation strategy would be to invalidate only this specific snapshot's mount state.
      queryClient.invalidateQueries({
        queryKey: klosetKeys.mounts(),
      });
    },
  });
};

export const useDeleteMount = ({ api, repository }: useCreateMountParams) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (mountId: string) => {
      if (!api.mount) {
        throw new Error("Mount API is not available");
      }
      return api.mount.delete({
        repository,
        mountId,
      });
    },
    onSuccess: () => {
      // XXX: better invalidation strategy would be to invalidate only this specific snapshot's mount state.
      queryClient.invalidateQueries({
        queryKey: klosetKeys.mounts(),
      });
    },
  });
};
