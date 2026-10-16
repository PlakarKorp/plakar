import type { IntegrationCardTypeKey } from "@plakar-ui/common/components";

import type { IntegrationType } from "~/api/integrations/schema.ts";

export const CardToOssApi: Record<
  IntegrationCardTypeKey,
  IntegrationType | undefined
> = {
  storage: "storage",
  source: "source",
  destination: "destination",
  secret_provider: undefined, // no secret provider for OSS
  inventory: undefined, // no inventory for OSS
};

export const OssApiToCard: Record<IntegrationType, IntegrationCardTypeKey> = {
  storage: "storage",
  source: "source",
  destination: "destination",
};
