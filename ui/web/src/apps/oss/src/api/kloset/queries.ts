import { klosetKeys } from "@plakar-ui/common/api/kloset";
import { queryOptions } from "@tanstack/react-query";

import { getRepositoryFacets, getSnapshotIntegrity } from "./api";

export const repositoryFacetsQueryOptions = () =>
  queryOptions({
    queryKey: [...klosetKeys.all(), "repository.facets"],
    // A full pass over the snapshots: no need to redo it on every mount.
    staleTime: 60 * 1000,
    queryFn: getRepositoryFacets,
  });

export const snapshotIntegrityQueryOptions = (snapshot: string) =>
  queryOptions({
    queryKey: [...klosetKeys.all(), "snapshot.integrity", snapshot],
    queryFn: () => getSnapshotIntegrity(snapshot),
    // Polled while the server reads the snapshot back; final once it is done.
    refetchInterval: (query) =>
      query.state.data?.item.status === "running" ? 1500 : false,
    staleTime: Infinity,
  });
