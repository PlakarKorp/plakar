import {
  Button,
  Icon,
  IconButton,
  Link,
  Tag,
  Typography,
} from "@plakar-ui/ui/components";
import { ComponentProps, useState } from "react";
import { LinkProps } from "react-aria-components";

import { IntegrationInstallationStatuses } from "./integration-card.constants";
import { IntegrationStageTag } from "./integration-version-tag";
import { RemoveIntegrationModal } from "./remove-integration-modal";
import { getIntegrationIcon, hasIntegrationIcon } from "../../api/enrich";

type IntegrationInstallationStatus =
  (typeof IntegrationInstallationStatuses)[number];

const INTEGRATION_TYPE_TAGS = [
  { key: "storage", label: "Storage" },
  { key: "source", label: "Source" },
  { key: "destination", label: "Destination" },
  { key: "secret_provider", label: "Secret provider" },
  { key: "inventory", label: "Inventory" },
] as const;

type IntegrationTypeTag = (typeof INTEGRATION_TYPE_TAGS)[number];
export type IntegrationCardTypeKey = IntegrationTypeTag["key"];

export type IntegrationCardProps = {
  name: string;
  icon?: string;
  displayName: string;
  version: string;
  stage: string;
  installationStatus: IntegrationInstallationStatus;
  types: {
    source?: boolean;
    destination?: boolean;
    storage?: boolean;
    secret_provider?: boolean;
    inventory?: boolean;
  };
  tags: string[];
  description: string;
  documentationProps?: ComponentProps<typeof Button>;
  onInstall?: () => void;
  onRemove?: () => void;
  onUpgrade?: () => void;
  latestVersion?: string;
  pending?: boolean;
  isDisabled?: boolean;
  getTypeHref?: (
    typeKey: IntegrationCardTypeKey
  ) => LinkProps["href"] | undefined;
  highlightedTypes?: Array<IntegrationCardTypeKey>;
  quickActions?: Array<{ label: string; href: LinkProps["href"] }>;
};

export function IntegrationCard({
  name,
  icon,
  displayName,
  version,
  stage,
  installationStatus,
  types,
  tags,
  description,
  documentationProps,
  onInstall,
  onRemove,
  onUpgrade,
  latestVersion,
  pending = false,
  isDisabled = false,
  getTypeHref,
  highlightedTypes,
  quickActions,
}: IntegrationCardProps) {
  return (
    <div className="border-secondary bg-primary flex flex-col gap-4 rounded-xl border p-4">
      {/* Top */}
      <div className="flex gap-2">
        <div className="bg-tertiary flex h-11 w-11 shrink-0 items-center justify-center rounded-lg p-2">
          <IntegrationIcon name={name} icon={icon} />
        </div>

        <div className="flex flex-col gap-1">
          <Typography
            as="h2"
            variant="heading"
            weight="medium"
            size="sm"
            className="text-primary leading-none break-all"
          >
            {displayName}
          </Typography>
          <div className="flex items-center gap-1">
            <Typography className="text-tertiary leading-none break-all">
              {version}
            </Typography>
            <IntegrationStageTag stage={stage} />
          </div>
        </div>

        <div className="ml-auto">
          <InstallationStatus
            status={installationStatus}
            onInstall={onInstall}
            pending={pending}
            isDisabled={isDisabled}
          />
        </div>
      </div>

      {/* Content */}
      <div className="flex flex-col gap-3">
        <div className="flex flex-wrap gap-2">
          {INTEGRATION_TYPE_TAGS.filter(({ key }) => types[key]).map(
            ({ key, label }) => (
              <Tag
                key={key}
                variant={
                  highlightedTypes?.includes(key)
                    ? "blue_borderless"
                    : "default"
                }
                radius="sqrt"
                href={getTypeHref?.(key)}
              >
                {label}
              </Tag>
            )
          )}
          {tags.map((tag) => (
            <Tag key={tag} variant="transparent" radius="rounded">
              {tag}
            </Tag>
          ))}
        </div>
        <Typography className="text-secondary overflow-auto" size="sm">
          {description}
        </Typography>
        {quickActions && quickActions.length > 0 && (
          <div className="flex flex-wrap gap-x-4 gap-y-1">
            {quickActions.map(({ label, href }) => (
              <Link
                key={label}
                href={href}
                className="inline-flex items-center gap-1 text-sm font-medium"
              >
                <Icon type="plus" size="sm" />
                {label}
              </Link>
            ))}
          </div>
        )}
      </div>

      {/* Actions */}
      <div className="mt-auto flex gap-2 text-center">
        {documentationProps && (
          <Button
            variant="secondary"
            className="flex-1"
            {...documentationProps}
          >
            Read documentation
          </Button>
        )}

        {installationStatus === "installed" && onUpgrade && (
          <Button
            variant="accent-tertiary"
            className="flex-1"
            leftIcon="circle-up"
            onPress={onUpgrade}
            isDisabled={pending || isDisabled}
            isPending={pending}
          >
            {latestVersion ? `Upgrade to ${latestVersion}` : "Upgrade"}
          </Button>
        )}

        {installationStatus === "installed" && onRemove && (
          <RemoveIntegrationButton
            displayName={displayName}
            version={version}
            onRemove={onRemove}
            pending={pending}
            isDisabled={isDisabled}
          />
        )}
      </div>
    </div>
  );
}

type RemoveIntegrationButtonProps = {
  displayName: string;
  version: string;
  onRemove: () => void;
  pending: boolean;
  isDisabled: boolean;
};

function RemoveIntegrationButton({
  displayName,
  version,
  onRemove,
  pending,
  isDisabled,
}: RemoveIntegrationButtonProps) {
  const [isConfirming, setIsConfirming] = useState(false);

  return (
    <>
      <IconButton
        aria-label="Remove"
        tooltip="Remove"
        variant="tertiary"
        className="ml-auto"
        icon="trash"
        onPress={() => setIsConfirming(true)}
        isDisabled={pending || isDisabled}
        isPending={pending}
      />
      <RemoveIntegrationModal
        displayName={displayName}
        version={version}
        isOpen={isConfirming}
        onOpenChange={setIsConfirming}
        onConfirm={onRemove}
      />
    </>
  );
}

type IntegrationIconProps = {
  name: string;
  icon?: string;
};

function IntegrationIcon({ name, icon }: IntegrationIconProps) {
  if (!hasIntegrationIcon(name) && icon) {
    return (
      <img
        src={icon}
        className="h-full w-full object-cover object-center"
        alt=""
      />
    );
  }

  return <Icon type={getIntegrationIcon(name)} size="2xl" />;
}

type InstallationStatusProps = {
  status: IntegrationInstallationStatus;
  onInstall: (() => void) | undefined;
  pending: boolean;
  isDisabled: boolean;
};

function InstallationStatus({
  status,
  onInstall,
  pending,
  isDisabled,
}: InstallationStatusProps) {
  return {
    installed: (
      <div className="flex items-center gap-1">
        <Icon type="success-circle" size="md" className="text-accent" />
        <Typography weight="medium" size="sm" className="text-accent">
          Installed
        </Typography>
      </div>
    ),
    "not-installed": (
      <Button
        leftIcon="download"
        isDisabled={!onInstall || pending || isDisabled}
        onPress={onInstall}
        isPending={pending}
        className="w-full"
      >
        Install
      </Button>
    ),
    "built-in": (
      <div className="flex items-center gap-1">
        <Icon type="success-circle" size="md" className="text-accent" />
        <Typography weight="medium" size="sm" className="text-accent">
          Built-in
        </Typography>
      </div>
    ),
  }[status];
}
