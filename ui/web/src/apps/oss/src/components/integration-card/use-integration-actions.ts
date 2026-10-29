import { addToast } from "@plakar-ui/common/components";

import {
  useInstallIntegration,
  useRemoveIntegration,
} from "~/api/integrations/mutations";
import type { Integration } from "~/api/integrations/schema";

function toastMessages(messages: { message: string }[]) {
  addToast({
    content: {
      children: messages.map((message) => message.message).join(" - "),
      variant: "success",
    },
  });
}

function toastError(error: Error) {
  addToast({
    content: { children: error.message, variant: "error" },
  });
}

export function isUpgradable({ installation, latest_version }: Integration) {
  return (
    installation.status === "installed" &&
    installation.version !== undefined &&
    installation.version !== latest_version
  );
}

export function useIntegrationActions(
  integration: Integration,
  installDisabled: boolean
) {
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
                if (successMessage) {
                  addToast({
                    content: { children: successMessage, variant: "success" },
                  });
                } else {
                  toastMessages(messages);
                }
              },
              onError: toastError,
            }
          );
        }
      : undefined;

  const onRemove = () => {
    uninstall.mutate(
      { integrationId: integration.id },
      {
        onSuccess: ({ messages }) => toastMessages(messages),
        onError: toastError,
      }
    );
  };

  return {
    onInstall: createInstallLatestVersion(),
    onUpgrade: isUpgradable(integration)
      ? createInstallLatestVersion(
          `${integration.display_name} upgraded to ${integration.latest_version}.`
        )
      : undefined,
    onRemove,
    pending: install.isPending || uninstall.isPending,
  };
}
