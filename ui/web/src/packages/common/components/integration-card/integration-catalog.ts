import {
  infiniteQueryOptions,
  QueryKey,
  queryOptions,
} from "@tanstack/react-query";
import { ReactNode } from "react";
import { LinkProps } from "react-aria-components";

import { IntegrationInstallationStatus } from "./integration-card.constants";
import { PaginationParams } from "../../api/pagination";

export type IntegrationCatalogItem = {
  id: string;
  name: string;
  icon?: string;
  display_name: string;
  latest_version: string;
  installation: {
    status: IntegrationInstallationStatus;
    version?: string;
  };
};

export type IntegrationCatalogFilters<Type extends string> = {
  search: string;
  types: Type[] | undefined;
};

export type IntegrationCatalogListParams<
  Type extends string,
  Status extends string,
> = PaginationParams &
  IntegrationCatalogFilters<Type> & {
    installationStatus?: Status;
  };

export type IntegrationCatalog<
  T extends IntegrationCatalogItem,
  Type extends string,
  Status extends string,
> = {
  list: (
    params: IntegrationCatalogListParams<Type, Status>
  ) => Promise<{ total: number; items: T[] }>;
  listsQueryKey: QueryKey;
  refetchInterval?: number;
  detailHref?: (integration: T) => LinkProps["href"];
  renderTile: (integration: T) => ReactNode;
  renderCard: (integration: T) => ReactNode;
};

export function isUpgradable({
  installation,
  latest_version,
}: IntegrationCatalogItem) {
  return (
    installation.status === "installed" &&
    installation.version !== undefined &&
    installation.version !== latest_version
  );
}

export function integrationListQueryOptions<
  T extends IntegrationCatalogItem,
  Type extends string,
  Status extends string,
>(
  catalog: IntegrationCatalog<T, Type, Status>,
  params: IntegrationCatalogListParams<Type, Status>
) {
  return queryOptions({
    queryKey: [...catalog.listsQueryKey, params],
    queryFn: () => catalog.list(params),
    refetchInterval: catalog.refetchInterval,
  });
}

export function integrationInfiniteListQueryOptions<
  T extends IntegrationCatalogItem,
  Type extends string,
  Status extends string,
>(
  catalog: IntegrationCatalog<T, Type, Status>,
  params: Omit<IntegrationCatalogListParams<Type, Status>, "page">
) {
  return infiniteQueryOptions({
    queryKey: [...catalog.listsQueryKey, "infinite", params],
    queryFn: ({ pageParam }) => catalog.list({ ...params, page: pageParam }),
    initialPageParam: 0,
    getNextPageParam: (lastPage, pages) =>
      pages.length * params.perPage < lastPage.total ? pages.length : undefined,
    refetchInterval: catalog.refetchInterval,
  });
}

export const INTEGRATION_SHELF_PAGE_SIZE = 50;
export const INTEGRATION_GRID_PAGE_SIZE = 24;

export function integrationShelfQueryOptions<
  T extends IntegrationCatalogItem,
  Type extends string,
  Status extends string,
>(
  catalog: IntegrationCatalog<T, Type, Status>,
  status: Status,
  filters: IntegrationCatalogFilters<Type>
) {
  return integrationInfiniteListQueryOptions(catalog, {
    ...filters,
    installationStatus: status,
    perPage: INTEGRATION_SHELF_PAGE_SIZE,
  });
}

export function integrationGridQueryOptions<
  T extends IntegrationCatalogItem,
  Type extends string,
  Status extends string,
>(
  catalog: IntegrationCatalog<T, Type, Status>,
  status: Status,
  filters: IntegrationCatalogFilters<Type>,
  page: number
) {
  return integrationListQueryOptions(catalog, {
    ...filters,
    installationStatus: status,
    page,
    perPage: INTEGRATION_GRID_PAGE_SIZE,
  });
}
