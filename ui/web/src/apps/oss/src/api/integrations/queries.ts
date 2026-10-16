import { queryOptions } from "@tanstack/react-query";

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
