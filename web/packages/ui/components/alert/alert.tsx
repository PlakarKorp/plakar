import clsx from "clsx";
import { PropsWithChildren } from "react";

import { Icon, IconType, Typography } from "..";

type AlertProps = PropsWithChildren<{
  variant?: "default" | "brand" | "success" | "warning" | "error";
  title?: string;
  className?: string;
  actions?: React.ReactNode;
  size?: "sm" | "md";
  /** Overrides the variant's icon. */
  icon?: IconType;
}>;

export function Alert({
  variant = "default",
  title,
  className,
  children,
  actions,
  size = "md",
  icon,
}: AlertProps) {
  const variantIcon = (
    {
      default: "info-circle",
      brand: "info-circle",
      success: "success-circle",
      warning: "warning-triangle",
      error: "error-circle",
    } as const
  )[variant];

  return (
    <div
      className={clsx(
        "flex flex-wrap items-center justify-between gap-2 gap-y-3 overflow-auto border-1",
        {
          default: "border-primary bg-secondary text-primary",
          brand: "border-accent bg-accent-tertiary text-accent",
          success: "border-success bg-success-tertiary text-success",
          warning: "border-warning bg-warning-tertiary text-warning",
          error: "border-error bg-error-tertiary text-error",
        }[variant],
        {
          sm: "rounded-lg p-2",
          md: "rounded-xl p-3",
        }[size],
        className
      )}
    >
      <div
        className={clsx(
          "flex flex-1",
          {
            sm: "gap-2",
            md: "gap-3",
          }[size]
        )}
      >
        <Icon type={icon ?? variantIcon} size="md" />

        <div className="flex flex-1 flex-col gap-1">
          <Typography weight="medium" size={size} className="leading-none">
            {title}
          </Typography>
          {children && (
            <Typography
              as="div"
              size={size === "sm" ? "xs" : "sm"}
              className="text-secondary"
            >
              {children}
            </Typography>
          )}
        </div>
      </div>

      {actions && (
        <div className="flex flex-wrap gap-x-3 gap-y-1">{actions}</div>
      )}
    </div>
  );
}
