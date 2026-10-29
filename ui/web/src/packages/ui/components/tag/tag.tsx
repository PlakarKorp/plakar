import clsx from "clsx";
import { PropsWithChildren } from "react";
import { Link, LinkProps } from "react-aria-components";
import {
  OverflowList,
  OverflowListProps,
} from "react-responsive-overflow-list";

import { Icon, IconType, Typography } from "..";
import { TagVariant, TagRadius, TagSize } from "./tag.constants";
import { TypographySize } from "../typography/typography.constants";

type TagProps = PropsWithChildren<{
  variant?: (typeof TagVariant)[number];
  radius?: (typeof TagRadius)[number];
  size?: (typeof TagSize)[number];
  leftIcon?: IconType;
  rightIcon?: IconType;
  className?: string;
  href?: LinkProps["href"];
  onPress?: LinkProps["onPress"];
}>;

export function Tag({
  variant = "default",
  radius = "rounded",
  size = "md",
  leftIcon,
  rightIcon,
  children,
  className,
  href,
  onPress,
}: TagProps) {
  const isInteractive = href !== undefined || onPress !== undefined;
  const Element = isInteractive ? Link : "div";
  const linkProps = isInteractive ? { href, onPress } : {};

  return (
    <Element
      {...linkProps}
      className={clsx(
        isInteractive &&
          "data-focused:outline-focus cursor-pointer outline-offset-2 data-hovered:underline",
        // Variant
        {
          default: "border-secondary bg-tertiary text-secondary",
          default_borderless:
            "bg-tertiary text-secondary border-background-tertiary",
          heavy: "border-accent-heavy bg-accent-heavy text-on-accent",
          brand: "border-accent bg-accent text-on-brand",
          blue: "border-accent bg-accent-tertiary text-accent",
          blue_borderless:
            "bg-accent-tertiary text-accent border-background-accent-tertiary",
          green: "border-success bg-success-tertiary text-success",
          yellow: "border-warning bg-warning-tertiary text-warning",
          yellow_borderless:
            "bg-warning-tertiary text-warning border-background-warning-tertiary",
          red: "border-error bg-error-tertiary text-error",
          transparent: "border-secondary text-secondary",
          source: "border-source bg-source-accent text-on-brand",
          source_secondary: "bg-source-tertiary text-source border-source",
          destination: "border-destination bg-destination-accent text-on-brand",
          destination_secondary:
            "bg-destination-tertiary text-destination border-destination",
          store: "border-store bg-store-accent text-on-brand",
          store_secondary: "bg-store-tertiary text-store border-store",
          secret_provider:
            "border-secret-provider bg-secret-provider-accent text-on-brand",
          disabled: "border-secondary text-disable",
        }[variant],
        // Radius
        {
          rounded: "rounded-2xl",
          sqrt: "rounded-sm",
        }[radius],
        // Size
        {
          xs: "h-4 gap-0.5 px-1 py-0.25",
          sm: "h-5 gap-1 px-1.5 py-0.25",
          md: "h-6.5 gap-1.5 px-2 py-0.5",
        }[size],
        // Common styles
        "inline-flex items-center justify-center border",
        className
      )}
    >
      {leftIcon && <Icon type={leftIcon} size="sm" />}
      {children !== null &&
        children !== undefined &&
        children !== "" &&
        children !== false && (
          <Typography
            as="div"
            size={
              {
                xs: "xs",
                sm: "xs",
                md: "sm",
              }[size] as TypographySize
            }
            className="truncate text-center"
          >
            {children}
          </Typography>
        )}
      {rightIcon && <Icon type={rightIcon} size="sm" />}
    </Element>
  );
}

export type TagOverflowItem = { label: string; leftIcon?: IconType };

type TagOverflowListProps = Pick<
  OverflowListProps<TagOverflowItem>,
  "maxRows" | "className"
> &
  Omit<TagProps, "children" | "leftIcon" | "rightIcon"> & {
    items?: (string | TagOverflowItem)[];
  };

export function TagOverflowList({
  items = [],
  maxRows = 1,
  className = "gap-1",
  ...tagProps
}: TagOverflowListProps) {
  return (
    <OverflowList
      items={items.map((item) =>
        typeof item === "string" ? { label: item } : item
      )}
      renderItem={({ label, leftIcon }) => (
        <Tag {...tagProps} leftIcon={leftIcon}>
          {label}
        </Tag>
      )}
      renderOverflow={(items) => (
        <div className="inline-flex">
          <Tag {...tagProps}>+{items.length}</Tag>
        </div>
      )}
      maxRows={maxRows}
      className={clsx("pr-px", className)}
    />
  );
}
