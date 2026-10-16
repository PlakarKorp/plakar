import { IconType, Typography } from "@plakar-ui/ui/components";
import clsx from "clsx";
import { ComponentProps, ElementType, ReactNode } from "react";

import { IconBadge, IconBadgeType } from "../icon-badge/icon-badge";

type QuickActionCardProps<T extends ElementType> = {
  as?: T;
  title: string;
  description: string;
  icon: IconType;
  iconBadgeType?: IconBadgeType;
  loading?: boolean;
  interactive?: boolean;
  isDisabled?: boolean;
  action?: ReactNode;
};

type QuickActionCardPropsWithAsProps<T extends ElementType> =
  QuickActionCardProps<T> &
    Omit<ComponentProps<T>, keyof QuickActionCardProps<T>>;

export function QuickActionCard<T extends ElementType>({
  as,
  title,
  description,
  icon,
  iconBadgeType = "accent",
  loading = false,
  interactive = true,
  isDisabled = false,
  action,
  ...props
}: QuickActionCardPropsWithAsProps<T>) {
  const Tag = as ?? "div";

  return (
    <Tag
      className={clsx(
        "bg-primary text-primary rounded-lg border px-6 py-4 text-left",
        interactive && !isDisabled
          ? "border-primary hover:border-accent group cursor-pointer hover:shadow-lg"
          : "border-secondary",
        isDisabled && "cursor-not-allowed opacity-50"
      )}
      {...(isDisabled && { isDisabled })}
      {...props}
    >
      <div className="flex h-full flex-1 items-center justify-between gap-6">
        <div className="flex h-full flex-col">
          <Typography as="div" weight="medium" size="md">
            {title}
          </Typography>
          <Typography variant="body" size="sm" className="text-tertiary">
            {description}
          </Typography>
        </div>

        <div className="flex shrink-0 items-center gap-3">
          {action}
          <IconBadge
            icon={icon}
            type={iconBadgeType}
            loading={loading}
            size="md"
            className="outline-background-primary transition-all duration-200 ease-out group-hover:scale-[1.1] group-hover:outline-1"
          />
        </div>
      </div>
    </Tag>
  );
}
