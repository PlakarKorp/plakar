import { keepPreviousData, useQueries, useQuery } from "@tanstack/react-query";
import { useState } from "react";

import {
  IntegrationCatalog,
  IntegrationCatalogFilters,
  IntegrationCatalogItem,
  integrationListQueryOptions,
} from "./integration-catalog";

export type IntegrationCounts<Status extends string> = {
  byStatus: Partial<Record<Status, number>>;
  unfiltered: number | undefined;
};

export function useIntegrationCounts<
  T extends IntegrationCatalogItem,
  Type extends string,
  Status extends string,
>(
  catalog: IntegrationCatalog<T, Type, Status>,
  statuses: readonly Status[],
  filters: IntegrationCatalogFilters<Type>
): IntegrationCounts<Status> {
  const totals = useQueries({
    queries: statuses.map((installationStatus) => ({
      ...integrationListQueryOptions(catalog, {
        ...filters,
        installationStatus,
        page: 0,
        perPage: 1,
      }),
    })),
  });
  const { data: unfiltered } = useQuery({
    ...integrationListQueryOptions(catalog, {
      search: "",
      types: undefined,
      page: 0,
      perPage: 1,
    }),
    placeholderData: keepPreviousData,
  });

  const [lastKnown, setLastKnown] = useState<Partial<Record<Status, number>>>(
    {}
  );
  const byStatus: Partial<Record<Status, number>> = {};
  statuses.forEach((status, index) => {
    const total = totals[index]?.data?.total ?? lastKnown[status];
    if (total !== undefined) {
      byStatus[status] = total;
    }
  });
  const isSettled = totals.every(({ data }) => data !== undefined);
  if (
    isSettled &&
    statuses.some((status) => byStatus[status] !== lastKnown[status])
  ) {
    setLastKnown(byStatus);
  }

  return { byStatus, unfiltered: unfiltered?.total };
}

export function sumCounts(...counts: Array<number | undefined>) {
  return counts.some((count) => count === undefined)
    ? undefined
    : counts.reduce<number>((total, count) => total + (count ?? 0), 0);
}
