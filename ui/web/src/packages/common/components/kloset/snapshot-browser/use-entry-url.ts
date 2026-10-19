import { useQuery } from "@tanstack/react-query";

import {
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
}: {
  snapshot: string;
  source: number;
  path: string;
  enabled?: boolean;
}) {
  const { api, repository } = useKlosetAPI();
  const query = useQuery({
    ...SnapshotReaderURLQueryOptions(api, {
      repository,
      snapshot,
      source,
      path,
    }),
    enabled,
  });
  const getUrl = query.data?.getUrl ?? undefined;
  return {
    url: getUrl?.(),
    getUrl,
    isColdStorage: query.data?.status === "glacier",
    error: query.error,
  };
}
