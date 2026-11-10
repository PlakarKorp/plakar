import { klosetKeys } from "@plakar-ui/common/api/kloset";
import { queryOptions } from "@tanstack/react-query";

import { getRepositoryFacets } from "./api";

export const repositoryFacetsQueryOptions = () =>
  queryOptions({
    queryKey: [...klosetKeys.all(), "repository.facets"],
    // A full pass over the snapshots: no need to redo it on every mount.
    staleTime: 60 * 1000,
    queryFn: getRepositoryFacets,
  });
