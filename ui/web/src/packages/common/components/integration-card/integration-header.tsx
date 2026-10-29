import { Button, Icon, Typography } from "@plakar-ui/ui/components";

import { IntegrationActionsMenu } from "./integration-actions-menu";
import {
  IntegrationInstallationStatus,
  IntegrationTypeFlags,
} from "./integration-card.constants";
import { IntegrationIcon } from "./integration-icon";
import { IntegrationTypeTags } from "./integration-type-tags";
import { IntegrationStageTag } from "./integration-version-tag";

type IntegrationHeaderProps = {
  name: string;
  icon?: string;
  displayName: string;
  description: string;
  version: string;
  stage: string;
  types: IntegrationTypeFlags;
  status: IntegrationInstallationStatus;
  latestVersion?: string;
  onInstall?: () => void;
  onUpgrade?: () => void;
  onRemove?: () => void;
  pending?: boolean;
  isDisabled?: boolean;
};

export function IntegrationHeader({
  name,
  icon,
  displayName,
  description,
  version,
  stage,
  types,
  status,
  latestVersion,
  onInstall,
  onUpgrade,
  onRemove,
  pending = false,
  isDisabled = false,
}: IntegrationHeaderProps) {
  return (
    <header className="flex flex-wrap items-start gap-5">
      <span className="bg-tertiary border-secondary flex size-16 shrink-0 items-center justify-center overflow-hidden rounded-2xl border p-3">
        <IntegrationIcon name={name} icon={icon} />
      </span>

      <div className="flex min-w-0 flex-[1_1_22rem] flex-col gap-2">
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
          <Typography
            as="h1"
            variant="heading"
            weight="semibold"
            className="text-primary"
          >
            {displayName}
          </Typography>
          <InstallationStatus status={status} />
        </div>
        <Typography size="sm" className="text-secondary">
          {description}
        </Typography>
        <div className="flex flex-wrap items-center gap-2">
          <Typography as="span" size="xs" monospace className="text-tertiary">
            {version}
          </Typography>
          {stage !== "stable" && <IntegrationStageTag stage={stage} />}
          <IntegrationTypeTags types={types} />
        </div>
      </div>

      <div className="flex items-center gap-2">
        {status === "not-installed" && (
          <Button
            leftIcon="plus"
            onPress={onInstall}
            isPending={pending}
            isDisabled={!onInstall || pending || isDisabled}
          >
            Install
          </Button>
        )}
        {latestVersion && (
          <Button
            leftIcon="arrow-up"
            onPress={onUpgrade}
            isPending={pending}
            isDisabled={!onUpgrade || pending || isDisabled}
          >
            Update to {latestVersion}
          </Button>
        )}
        <IntegrationActionsMenu
          displayName={displayName}
          version={version}
          onRemove={onRemove}
          pending={pending}
          isDisabled={isDisabled}
          variant="secondary"
        />
      </div>
    </header>
  );
}

function InstallationStatus({
  status,
}: {
  status: IntegrationInstallationStatus;
}) {
  if (status === "not-installed") {
    return null;
  }

  return (
    <Typography
      as="span"
      size="sm"
      weight="medium"
      className="text-success flex items-center gap-1.5"
    >
      <Icon type="check" size="sm" />
      {status === "built-in" ? "Built-in" : "Installed"}
    </Typography>
  );
}
