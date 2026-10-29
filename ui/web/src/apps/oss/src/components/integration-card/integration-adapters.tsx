import {
  IntegrationCard,
  IntegrationHeader,
  IntegrationTile,
} from "@plakar-ui/common/components";

import type { Integration } from "~/api/integrations/schema";
import {
  isUpgradable,
  useIntegrationActions,
} from "~/components/integration-card/use-integration-actions";

type IntegrationAdapterProps = {
  integration: Integration;
  installDisabled: boolean;
};

function detailHref(integration: Integration) {
  return {
    to: "/integrations/$integrationId",
    params: { integrationId: integration.id },
  } as const;
}

export function IntegrationTileAdapter({
  integration,
  installDisabled,
}: IntegrationAdapterProps) {
  const { onUpgrade, onRemove, pending } = useIntegrationActions(
    integration,
    installDisabled
  );
  const isBuiltIn = integration.installation.status === "built-in";

  return (
    <IntegrationTile
      name={integration.name}
      icon={integration.icon}
      displayName={integration.display_name}
      version={integration.installation.version ?? integration.latest_version}
      stage={integration.stage}
      isBuiltIn={isBuiltIn}
      detailHref={detailHref(integration)}
      latestVersion={
        isUpgradable(integration) ? integration.latest_version : undefined
      }
      onUpgrade={onUpgrade}
      onRemove={isBuiltIn ? undefined : onRemove}
      pending={pending}
      isDisabled={installDisabled}
    />
  );
}

export function IntegrationCardAdapter({
  integration,
  installDisabled,
}: IntegrationAdapterProps) {
  const { onInstall, onUpgrade, onRemove, pending } = useIntegrationActions(
    integration,
    installDisabled
  );
  const { status } = integration.installation;

  return (
    <IntegrationCard
      name={integration.name}
      icon={integration.icon}
      displayName={integration.display_name}
      version={integration.installation.version ?? integration.latest_version}
      stage={integration.stage}
      description={integration.description}
      types={integration.types}
      status={status}
      detailHref={detailHref(integration)}
      latestVersion={
        isUpgradable(integration) ? integration.latest_version : undefined
      }
      onInstall={onInstall}
      onUpgrade={onUpgrade}
      onRemove={status === "installed" ? onRemove : undefined}
      pending={pending}
      isDisabled={installDisabled}
    />
  );
}

export function IntegrationHeaderAdapter({
  integration,
  installDisabled,
}: IntegrationAdapterProps) {
  const { onInstall, onUpgrade, onRemove, pending } = useIntegrationActions(
    integration,
    installDisabled
  );
  const { status } = integration.installation;

  return (
    <IntegrationHeader
      name={integration.name}
      icon={integration.icon}
      displayName={integration.display_name}
      description={integration.description}
      version={integration.installation.version ?? integration.latest_version}
      stage={integration.stage}
      types={integration.types}
      status={status}
      latestVersion={
        isUpgradable(integration) ? integration.latest_version : undefined
      }
      onInstall={onInstall}
      onUpgrade={onUpgrade}
      onRemove={status === "installed" ? onRemove : undefined}
      pending={pending}
      isDisabled={installDisabled}
    />
  );
}
