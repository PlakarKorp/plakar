import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { listIntegrationsQueryOptions } from "~/api/integrations/queries";
import type {
  IntegrationInstallationStatusFilter,
  IntegrationType,
} from "~/api/integrations/schema";

type CountedStatus = Extract<
  IntegrationInstallationStatusFilter,
  "installed" | "built-in" | "not-installed" | "upgradable"
>;

export type IntegrationCounts = Record<CountedStatus, number | undefined> & {
  unfiltered: number | undefined;
};

function sum(...counts: Array<number | undefined>) {
  return counts.some((count) => count === undefined)
    ? undefined
    : counts.reduce<number>((total, count) => total + (count ?? 0), 0);
}

export function shownCount(
  counts: IntegrationCounts,
  status: IntegrationInstallationStatusFilter | undefined
) {
  switch (status) {
    case undefined:
      return sum(counts.installed, counts["built-in"], counts["not-installed"]);
    case "installed":
      return sum(counts.installed, counts["built-in"]);
    default:
      return counts[status];
  }
}

function useStatusCount(
  installationStatus: CountedStatus | undefined,
  search: string,
  types: IntegrationType[] | undefined
) {
  const { data } = useQuery({
    ...listIntegrationsQueryOptions({
      search,
      types,
      installationStatus,
      page: 0,
      perPage: 1,
    }),
    placeholderData: keepPreviousData,
  });
  return data?.total;
}

export function useIntegrationCounts({
  search,
  types,
}: {
  search: string;
  types: IntegrationType[] | undefined;
}): IntegrationCounts {
  return {
    installed: useStatusCount("installed", search, types),
    "built-in": useStatusCount("built-in", search, types),
    "not-installed": useStatusCount("not-installed", search, types),
    upgradable: useStatusCount("upgradable", search, types),
    unfiltered: useStatusCount(undefined, "", undefined),
  };
}
