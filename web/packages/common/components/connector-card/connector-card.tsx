import {
  Hint,
  Icon,
  IconType,
  Link as LinkComponent,
  Tag,
  TextHighlight,
  Typography,
} from "@plakar-ui/ui/components";
import clsx from "clsx";
import { PropsWithChildren, ReactNode } from "react";
import { LinkProps as AriaLinkProps } from "react-aria-components";

import { type EnrichedResidency } from "../../api/enrich";
import { StorageTier } from "../../api/storage-tier";
import { IconBadge, IconBadgeType } from "../icon-badge/icon-badge";
import { ResidencyTag } from "../residency/residency";
import { getStorageTierIcon } from "../storage-tier/storage-tier";

type ConnectorCardProps = PropsWithChildren<{
  isFullWidth?: boolean;
}>;

function ConnectorCardRoot({
  isFullWidth = false,
  children,
}: ConnectorCardProps) {
  return (
    <div
      className={clsx(
        "px-6 py-4",
        "border-primary rounded-lg border",
        "bg-primary hover:bg-primary-hover",
        isFullWidth ? "w-full" : "w-fit"
      )}
    >
      {children}
    </div>
  );
}

export type ConnectorCardContentProps = {
  inventory: InventoryProps;
  resource: ResourceProps;
  connector: ConnectorProps;
  details?: ReactNode;
};

export function Content({
  inventory,
  resource,
  connector,
  details,
}: ConnectorCardContentProps) {
  return (
    <div className="flex items-center gap-3">
      <IconBadge
        type={connector.type}
        icon={resource.icon}
        size="md"
        className="my-2"
      />

      <div className="min-w-0">
        <div className="-mt-1 mb-1">
          {inventory ? (
            <Inventory {...inventory} size="xs" />
          ) : (
            <div className="h-4" />
          )}
        </div>

        <div className="flex flex-wrap items-center gap-x-2">
          <ResidencyTag residency={resource.residency} size="xs" />
          <Typography className="text-primary truncate" weight="medium">
            {resource.name}
          </Typography>
          <Icon type="chevron-right" size="xs" className="text-tertiary" />
          <Connector {...connector} size="xs" />
        </div>

        <Typography className="flex min-w-0 items-center gap-1.5" size="xs">
          <span className="text-secondary">{resource.resourceClass}</span>
          <span className="text-tertiary min-w-0 flex-1 truncate">
            {resource.urn}
          </span>
          {details && (
            <div className="text-xs">
              <Hint tooltip={details} underline="hover">
                more
              </Hint>
            </div>
          )}
        </Typography>
      </div>
    </div>
  );
}

type InventoryProps = {
  icon: IconType;
  iconBadgeType: IconBadgeType;
  name: string;
  search?: string;
  size?: "xs" | "sm" | "md";
  href?: AriaLinkProps["href"];
};

function Inventory({
  icon,
  iconBadgeType,
  name,
  search,
  size = "md",
  href,
}: InventoryProps) {
  return (
    <div
      className={clsx(
        "text-primary flex shrink-0 items-center",
        {
          xs: "gap-1.5",
          sm: "gap-1.5",
          md: "gap-2",
        }[size]
      )}
    >
      {size === "xs" || size === "sm" ? (
        <Icon type={icon} size="md" />
      ) : (
        <IconBadge type={iconBadgeType} icon={icon} size="sm" />
      )}
      <Link href={href}>
        <Typography weight="medium" size={size}>
          <TextHighlight text={name} search={search} />
        </Typography>
      </Link>
    </div>
  );
}

type ResourceProps = {
  icon: IconType;
  iconBadgeType: IconBadgeType;
  name: string;
  suffix?: ReactNode;
  resourceClass?: string;
  urn?: string;
  href?: AriaLinkProps["href"];
  search?: string;
  size?: "sm" | "md";
  tags: string[];
  residency: EnrichedResidency;
};

function Resource({
  icon,
  iconBadgeType,
  name,
  suffix,
  urn,
  resourceClass,
  href,
  search,
  size = "md",
  tags,
  residency,
}: ResourceProps) {
  return (
    <div className="flex min-w-0 flex-col gap-1">
      <div className="flex min-w-0 items-center gap-2">
        {size === "sm" ? (
          <Icon type={icon} size="md" className="shrink-0" />
        ) : (
          <IconBadge
            type={iconBadgeType}
            icon={icon}
            size="sm"
            className="shrink-0"
          />
        )}

        <div className="min-w-0">
          <div className="flex min-w-0 items-center gap-1.5">
            <ResidencyTag
              residency={residency}
              size={size === "sm" ? "xs" : "sm"}
            />
            <Link href={href}>
              <Typography
                className="min-w-0 truncate"
                weight="medium"
                size={size}
              >
                <TextHighlight text={name} search={search} />
              </Typography>
            </Link>
            {suffix && <div className="shrink-0">{suffix}</div>}
          </div>

          <Typography className="flex min-w-0 items-center gap-1.5" size="xs">
            {resourceClass && (
              <span className="text-secondary whitespace-nowrap">
                {resourceClass}
              </span>
            )}
            {urn && (
              <span className="text-tertiary min-w-0 flex-1 truncate">
                <TextHighlight text={urn} search={search} />
              </span>
            )}
          </Typography>
        </div>
      </div>

      {tags.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {tags.map((tag) => (
            <Tag key={tag} size="sm" radius="sqrt" variant="blue_borderless">
              {tag}
            </Tag>
          ))}
        </div>
      )}
    </div>
  );
}

type ConnectorProps = {
  type: "source" | "destination" | "store";
  name: string;
  integrationIcon?: IconType;
  search?: string;
  size?: "xs" | "sm" | "md";
  href?: AriaLinkProps["href"];
  isCurrent?: boolean;
  temperature?: StorageTier;
  truncate?: boolean;
};

function Connector({
  name,
  integrationIcon,
  search,
  size = "md",
  href,
  isCurrent = false,
  temperature,
  truncate = false,
}: ConnectorProps) {
  const storageTierIcon = getStorageTierIcon(
    temperature,
    ({ xs: "sm", sm: "sm", md: "md" } as const)[size]
  );

  return (
    <div
      className={clsx(
        "text-primary flex items-center text-left",
        truncate && "min-w-0",
        {
          xs: "gap-1.5",
          sm: "gap-1.5",
          md: "gap-2",
        }[size]
      )}
    >
      {integrationIcon && (
        <Icon
          type={integrationIcon}
          size={({ xs: "sm", sm: "md", md: "lg" } as const)[size]}
          className={clsx("text-secondary", truncate && "shrink-0")}
        />
      )}

      <Link href={href}>
        <Typography
          size={size}
          weight={isCurrent ? "normal" : "medium"}
          className={clsx(
            isCurrent && "italic",
            truncate && "min-w-0 truncate"
          )}
        >
          <TextHighlight text={name} search={search} />
        </Typography>
      </Link>

      {storageTierIcon &&
        (truncate ? (
          <span className="shrink-0">{storageTierIcon}</span>
        ) : (
          storageTierIcon
        ))}
    </div>
  );
}

type LinkProps = PropsWithChildren<{
  href?: AriaLinkProps["href"];
}>;

function Link({ href, children }: LinkProps) {
  if (!href) {
    return <>{children}</>;
  }

  return (
    <LinkComponent href={href} className="min-w-0" inheritColor>
      {children}
    </LinkComponent>
  );
}

export const ConnectorCard = Object.assign(ConnectorCardRoot, {
  Content,
  Inventory,
  Resource,
  Connector,
});

export const CONNECTOR_CARD_TRIGGER_HEIGHT = "min-h-21.5";
