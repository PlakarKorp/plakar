import { infiniteQueryOptions, queryOptions } from "@tanstack/react-query";

import { getIntegration, listIntegrations } from "./api";
import { integrationKeys } from "./keys";
import { ListIntegrationsParams } from "./schema";

export const getIntegrationQueryOptions = ({
  integration,
}: {
  integration: string;
}) =>
  queryOptions({
    queryKey: integrationKeys.detail(integration),
    queryFn: () => getIntegration({ integrationId: integration }),
    refetchInterval: 10 * 1000,
  });

export const listIntegrationsQueryOptions = (params: ListIntegrationsParams) =>
  queryOptions({
    queryKey: integrationKeys.list(params),
    queryFn: () => listIntegrations(params),
    refetchInterval: 10 * 1000,
  });

export const listIntegrationsInfiniteQueryOptions = (
  params: Omit<ListIntegrationsParams, "page">
) =>
  infiniteQueryOptions({
    queryKey: integrationKeys.infiniteList(params),
    queryFn: ({ pageParam }) =>
      listIntegrations({ ...params, page: pageParam }),
    initialPageParam: 0,
    getNextPageParam: (lastPage, pages) =>
      pages.length * params.perPage < lastPage.total ? pages.length : undefined,
    refetchInterval: 10 * 1000,
  });
