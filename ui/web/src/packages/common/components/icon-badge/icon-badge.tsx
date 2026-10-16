import { Icon, IconType, Skeleton, Spinner } from "@plakar-ui/ui/components";
import clsx from "clsx";

import { IconBadgeTypes } from "./icon-badge.constants";

export type IconBadgeType = (typeof IconBadgeTypes)[number];

type IconBadgeProps = {
  icon?: IconType;
  type?: IconBadgeType;
  size?: "sm" | "md";
  className?: string;
  loading?: boolean;
  isSkeleton?: boolean;
};

export function IconBadge({
  icon,
  type = "default",
  size = "md",
  className,
  loading = false,
  isSkeleton = false,
}: IconBadgeProps) {
  const iconSize = size === "md" ? "2xl" : "lg";

  return (
    <span
      className={clsx(
        "relative flex w-fit items-center justify-center rounded-md",
        {
          default: "bg-tertiary text-tertiary",
          accent: "bg-accent-tertiary text-accent",
          brand: "bg-accent text-on-brand",
          service: "text-on-brand bg-sky-600",
          source: "bg-source-tertiary text-source",
          destination: "bg-destination-tertiary text-destination",
          store: "bg-store-tertiary text-store",
          secret_provider: "bg-secret-provider-tertiary text-secret-provider",
          inventory: "bg-accent-tertiary text-accent",
          disabled: "bg-disable text-disable",
          error: "bg-error-tertiary text-error",
          warning: "bg-warning-tertiary text-warning",
        }[type],
        { sm: "p-2", md: "p-3" }[size],
        className
      )}
    >
      {isSkeleton ? (
        <Skeleton as="span" variant="icon" size={iconSize} />
      ) : loading ? (
        <Spinner size={iconSize} inheritColor />
      ) : (
        <>
          {icon && <Icon type={icon} size={iconSize} />}
          {icon === "plug-circle-xmark" && (
            <Icon
              type="error-circle"
              size="md"
              className={clsx(
                "text-error absolute",
                size === "md"
                  ? "right-3 bottom-[0.90625rem] h-3.5 w-3.5"
                  : "right-2 bottom-3 h-2.5 w-2.5"
              )}
            />
          )}
          {icon === "plug-circle-check" && (
            <Icon
              type="success-circle"
              size="md"
              className={clsx(
                "text-success absolute",
                size === "md"
                  ? "right-3 bottom-[0.90625rem] h-3.5 w-3.5"
                  : "right-2 bottom-3 h-2.5 w-2.5"
              )}
            />
          )}
        </>
      )}
    </span>
  );
}
