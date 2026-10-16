import {
  addToast,
  IntegrationCard,
  type IntegrationCardProps,
} from "@plakar-ui/common/components";

import {
  useInstallIntegration,
  useRemoveIntegration,
} from "~/api/integrations/mutations";
import type { Integration } from "~/api/integrations/schema";
import { CardToOssApi } from "~/components/integration-card/field-mapping";

const getTypeHref: IntegrationCardProps["getTypeHref"] = (displayKey) => {
  const wireKey = CardToOssApi[displayKey];
  if (!wireKey) return undefined;
  return {
    to: "/integrations",
    search: () => ({ page: 0, types: [wireKey] }),
  };
};

type IntegrationCardAdapterProps = Partial<IntegrationCardProps> & {
  integration: Integration;
  installDisabled?: boolean;
};

export function IntegrationCardAdapter({
  integration,
  installDisabled = false,
  ...props
}: IntegrationCardAdapterProps) {
  const install = useInstallIntegration();
  const uninstall = useRemoveIntegration();

  const createInstallLatestVersion = (successMessage?: string) =>
    !installDisabled && integration.installation.available
      ? () => {
          install.mutate(
            {
              integrationId: integration.id,
              version: integration.latest_version,
            },

            {
              onSuccess: ({ messages }) => {
                addToast({
                  content: {
                    children:
                      successMessage ??
                      messages.map((message) => message.message).join(" - "),
                    variant: "success",
                  },
                });
              },
              onError: (error) => {
                addToast({
                  content: { children: error.message, variant: "error" },
                });
              },
            }
          );
        }
      : undefined;

  const onInstall = createInstallLatestVersion();

  const onRemove = () => {
    uninstall.mutate(
      { integrationId: integration.id },
      {
        onSuccess: ({ messages }) => {
          addToast({
            content: {
              children: messages.map((message) => message.message).join(" - "),
              variant: "success",
            },
          });
        },
        onError: (error) => {
          addToast({
            content: { children: error.message, variant: "error" },
          });
        },
      }
    );
  };

  const { installation, latest_version, display_name } = integration;

  const canUpgrade =
    installation.status === "installed" &&
    installation.version !== undefined &&
    installation.version !== latest_version;

  const onUpgrade = canUpgrade
    ? createInstallLatestVersion(
        `${display_name} upgraded to ${latest_version}.`
      )
    : undefined;

  return (
    <IntegrationCard
      key={integration.id}
      name={integration.name}
      icon={integration.icon}
      displayName={integration.display_name}
      version={integration.latest_version}
      stage={integration.stage}
      installationStatus={integration.installation.status}
      types={{
        source: integration.types.source,
        destination: integration.types.destination,
        storage: integration.types.storage,
      }}
      tags={integration.tags}
      description={integration.description}
      onInstall={onInstall}
      onRemove={onRemove}
      pending={install.isPending || uninstall.isPending}
      getTypeHref={getTypeHref}
      onUpgrade={onUpgrade}
      latestVersion={latest_version}
      {...props}
    />
  );
}
