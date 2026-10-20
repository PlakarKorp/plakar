import { useQueryClient } from "@tanstack/react-query";
import { useAsyncList } from "react-stately";

import { RepositorySnapshotsQueryOptions } from "./queries";
import { SnapshotType } from "./schema";
import { useKlosetAPI } from "./use-kloset-api";

/**
 * Restore points of the repository, most recent first, loaded page by page.
 */
export function useSnapshotsAsyncList(perPage: number) {
  const { api, repository } = useKlosetAPI();
  const queryClient = useQueryClient();

  return useAsyncList<SnapshotType, number>({
    async load({ cursor }) {
      const page = cursor ?? 0;
      const { items, total } = await queryClient.fetchQuery(
        RepositorySnapshotsQueryOptions(api, {
          repository,
          pagination: { page, perPage },
        })
      );
      return {
        items,
        cursor: total > (page + 1) * perPage ? page + 1 : undefined,
      };
    },
  });
}
