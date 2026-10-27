import { UpgradeIndicator } from "@plakar-ui/ui/components";
import { useQuery } from "@tanstack/react-query";

import { listIntegrationsQueryOptions } from "~/api/integrations/queries";

export function UpgradableIntegrations() {
  const { data: total } = useQuery({
    ...listIntegrationsQueryOptions({
      page: 0,
      perPage: 1,
      installationStatus: "upgradable",
    }),
    select: (data) => data.total,
    refetchInterval: false,
  });

  if (!total) {
    return null;
  }

  return <UpgradeIndicator label={`${total} integrations to upgrade`} />;
}
