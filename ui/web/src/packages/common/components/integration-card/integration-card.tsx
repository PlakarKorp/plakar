import { Button, Icon, Link, Typography } from "@plakar-ui/ui/components";
import { ReactNode } from "react";
import { LinkProps } from "react-aria-components";

import { IntegrationActionsMenu } from "./integration-actions-menu";
import {
  IntegrationInstallationStatus,
  IntegrationTypeFlags,
} from "./integration-card.constants";
import { IntegrationIcon } from "./integration-icon";
import { IntegrationTypeTags } from "./integration-type-tags";
import { IntegrationStageTag } from "./integration-version-tag";

type IntegrationCardProps = {
  name: string;
  icon?: string;
  displayName: string;
  version: string;
  stage: string;
  description: string;
  types: IntegrationTypeFlags;
  status: IntegrationInstallationStatus;
  detailHref: LinkProps["href"];
  latestVersion?: string;
  onInstall?: () => void;
  onUpgrade?: () => void;
  onRemove?: () => void;
  pending?: boolean;
  isDisabled?: boolean;
};

export function IntegrationCard({
  name,
  icon,
  displayName,
  version,
  stage,
  description,
  types,
  status,
  detailHref,
  latestVersion,
  onInstall,
  onUpgrade,
  onRemove,
  pending = false,
  isDisabled = false,
}: IntegrationCardProps) {
  return (
    <article className="border-secondary bg-primary flex flex-col gap-3 rounded-xl border p-4">
      <div className="flex items-center gap-3">
        <span className="bg-tertiary flex size-12 shrink-0 items-center justify-center overflow-hidden rounded-xl p-2">
          <IntegrationIcon name={name} icon={icon} />
        </span>
        <span className="flex min-w-0 flex-1 flex-col gap-0.5">
          <Link
            href={detailHref}
            inheritColor
            className="text-primary truncate text-base font-semibold"
          >
            {displayName}
          </Link>
          <span className="flex items-center gap-1.5">
            {status === "installed" && (
              <Typography
                as="span"
                size="xs"
                weight="medium"
                className="text-success flex items-center gap-1"
              >
                <Icon type="check" size="xs" />
                Installed
              </Typography>
            )}
            <Typography as="span" size="xs" monospace className="text-tertiary">
              {version}
            </Typography>
            {stage !== "stable" && <IntegrationStageTag stage={stage} />}
          </span>
        </span>
        <IntegrationActionsMenu
          displayName={displayName}
          version={version}
          onRemove={onRemove}
          pending={pending}
          isDisabled={isDisabled}
          size="sm"
        />
      </div>
      <Typography size="sm" className="text-secondary line-clamp-2 min-h-10">
        {description}
      </Typography>
      <IntegrationTypeTags types={types} />
      <div className="mt-auto grid grid-cols-2 gap-2">
        <Button
          variant="secondary"
          href={detailHref}
          className="whitespace-nowrap"
        >
          Read documentation
        </Button>
        {status === "not-installed" ? (
          <Button
            leftIcon="plus"
            onPress={onInstall}
            isPending={pending}
            isDisabled={!onInstall || pending || isDisabled}
          >
            Install
          </Button>
        ) : latestVersion ? (
          <Button
            variant="accent-tertiary"
            leftIcon="arrow-up"
            onPress={onUpgrade}
            isPending={pending}
            isDisabled={!onUpgrade || pending || isDisabled}
            aria-label={`Update ${displayName} to ${latestVersion}`}
            className="min-w-0 [&>span:last-child]:min-w-0 [&>span:last-child]:truncate"
          >
            Update to {latestVersion}
          </Button>
        ) : (
          <CardStatus>
            {status === "built-in" ? "Built-in" : "Up to date"}
          </CardStatus>
        )}
      </div>
    </article>
  );
}

function CardStatus({ children }: { children: ReactNode }) {
  return (
    <Typography
      as="span"
      size="sm"
      weight="medium"
      className="text-success flex h-[38px] items-center justify-center gap-1.5"
    >
      <Icon type="check" size="sm" />
      {children}
    </Typography>
  );
}
