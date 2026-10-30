import { IntegrationCatalog } from "@plakar-ui/common/components";

import { listIntegrations } from "~/api/integrations/api";
import { integrationKeys } from "~/api/integrations/keys";
import type {
  Integration,
  IntegrationInstallationStatusFilter,
  IntegrationType,
} from "~/api/integrations/schema";
import {
  IntegrationCardAdapter,
  IntegrationTileAdapter,
} from "~/components/integration-card/integration-adapters";
import { integrationDetailHref } from "~/components/integration-card/integration-href";

export function integrationCatalog(
  installDisabled: boolean
): IntegrationCatalog<
  Integration,
  IntegrationType,
  IntegrationInstallationStatusFilter
> {
  return {
    list: listIntegrations,
    listsQueryKey: integrationKeys.lists(),
    refetchInterval: 10 * 1000,
    detailHref: integrationDetailHref,
    renderTile: (integration) => (
      <IntegrationTileAdapter
        integration={integration}
        installDisabled={installDisabled}
      />
    ),
    renderCard: (integration) => (
      <IntegrationCardAdapter
        integration={integration}
        installDisabled={installDisabled}
      />
    ),
  };
}
