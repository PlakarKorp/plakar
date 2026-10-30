import type { Integration } from "~/api/integrations/schema";

export function integrationDetailHref(integration: Integration) {
  return {
    to: "/integrations/$integrationId",
    params: { integrationId: integration.id },
  } as const;
}
