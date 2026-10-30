import { Button, Icon, Link, Typography } from "@plakar-ui/ui/components";
import clsx from "clsx";
import { LinkProps } from "react-aria-components";

import {
  IntegrationActionsMenu,
  IntegrationQuickAction,
} from "./integration-actions-menu";
import { IntegrationIcon } from "./integration-icon";
import { IntegrationStageTag } from "./integration-version-tag";

type IntegrationTileProps = {
  name: string;
  icon?: string;
  displayName: string;
  version: string;
  stage: string;
  isBuiltIn?: boolean;
  detailHref?: LinkProps["href"];
  latestVersion?: string;
  onUpgrade?: () => void;
  onRemove?: () => void;
  quickActions?: IntegrationQuickAction[];
  pending?: boolean;
  isDisabled?: boolean;
};

export function IntegrationTile({
  name,
  icon,
  displayName,
  version,
  stage,
  isBuiltIn = false,
  detailHref,
  latestVersion,
  onUpgrade,
  onRemove,
  quickActions,
  pending = false,
  isDisabled = false,
}: IntegrationTileProps) {
  const isUpgradeBlocked = !onUpgrade || pending || isDisabled;
  const hasActionsMenu =
    !!latestVersion ||
    !!detailHref ||
    !!onRemove ||
    (quickActions?.length ?? 0) > 0;

  return (
    <div
      className={clsx(
        "border-secondary bg-primary flex min-w-0 items-center gap-3 rounded-xl border py-2.5 pl-3",
        hasActionsMenu ? "pr-1.5" : "pr-3"
      )}
    >
      <span className="bg-tertiary flex size-8 shrink-0 items-center justify-center overflow-hidden rounded-lg p-1">
        <IntegrationIcon name={name} icon={icon} size="lg" />
      </span>
      <span className="flex min-w-0 flex-1 flex-col">
        {detailHref ? (
          <Link
            href={detailHref}
            inheritColor
            className="text-primary truncate text-sm font-semibold"
          >
            {displayName}
          </Link>
        ) : (
          <Typography
            as="span"
            size="sm"
            weight="semibold"
            className="text-primary truncate"
          >
            {displayName}
          </Typography>
        )}
        <span className="flex h-5 min-w-0 items-center gap-1.5">
          {!isBuiltIn && (
            <Icon type="check" size="xs" className="text-success shrink-0" />
          )}
          <Typography
            as="span"
            size="xs"
            monospace
            className="text-tertiary truncate"
          >
            {version}
          </Typography>
          {stage !== "stable" && <IntegrationStageTag stage={stage} />}
        </span>
      </span>
      {latestVersion ? (
        <Button
          size="xs"
          variant="accent-tertiary"
          leftIcon="arrow-up"
          onPress={onUpgrade}
          isPending={pending}
          isDisabled={isUpgradeBlocked}
          aria-label={`Update ${displayName} to ${latestVersion}`}
          className="shrink-0"
        >
          {latestVersion}
        </Button>
      ) : (
        <Typography
          as="span"
          size="xs"
          weight="medium"
          className="text-tertiary shrink-0"
        >
          {isBuiltIn ? "Built-in" : "Up to date"}
        </Typography>
      )}
      <IntegrationActionsMenu
        displayName={displayName}
        version={version}
        detailHref={detailHref}
        latestVersion={latestVersion}
        onUpgrade={onUpgrade}
        onRemove={onRemove}
        quickActions={quickActions}
        pending={pending}
        isDisabled={isDisabled}
        size="sm"
      />
    </div>
  );
}
