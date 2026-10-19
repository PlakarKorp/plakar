import { useQuery } from "@tanstack/react-query";

import {
  ReadFileParams,
  SnapshotReaderURLQueryOptions,
  useKlosetAPI,
} from "@plakar-ui/common/api/kloset";

/**
 * Returns the signed URL to read a file of a snapshot. It signs nothing until
 * enabled, so a gallery only signs the tiles that scroll into view.
 */
export function useEntryURL({
  snapshot,
  source,
  path,
  enabled = true,
  params,
}: {
  snapshot: string;
  source: number;
  path: string;
  enabled?: boolean;
  params?: ReadFileParams;
}) {
  const { api, repository } = useKlosetAPI();
  const query = useQuery({
    ...SnapshotReaderURLQueryOptions(api, {
      repository,
      snapshot,
      source,
      path,
      params,
    }),
    enabled,
  });
  return {
    url: query.data?.url ?? undefined,
    isColdStorage: query.data?.status === "glacier",
    isError: query.isError,
  };
}
