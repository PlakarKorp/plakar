import {
  RepositoryInfoQueryOptions,
  RepositorySnapshotsQueryOptions,
  useKlosetAPI,
} from "@plakar-ui/common/api/kloset";
import { useQuery } from "@tanstack/react-query";

export const STATS_NUMBER_OF_DAYS = 30;

function getDateNDaysAgo(numberOfDays: number) {
  const d = new Date();
  d.setDate(d.getDate() - numberOfDays);
  d.setHours(0, 0, 0, 0);
  return d;
}

export function useRepositoryStats() {
  const { api, repository } = useKlosetAPI();
  const { data } = useQuery(RepositoryInfoQueryOptions(api, { repository }));
  const { data: recentSnapshots } = useQuery(
    RepositorySnapshotsQueryOptions(api, {
      repository,
      pagination: {
        page: 0,
        perPage: 1,
      },
      since: getDateNDaysAgo(STATS_NUMBER_OF_DAYS),
    })
  );

  return {
    snapshots: data?.item.snapshots,
    recentTotal: recentSnapshots?.total,
  };
}
