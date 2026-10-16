import clsx from "clsx";
import { ComponentProps, PropsWithChildren } from "react";

import { Button, IconButton, Typography } from "..";
import { Icon } from "../icon/icon";

type ToasterProps = PropsWithChildren<{
  variant?: "default" | "brand" | "success" | "warning" | "error";
  title?: string;
  buttonProps?: ComponentProps<typeof Button>;
  closable?: boolean;
}>;

export function Toaster({
  variant = "default",
  title,
  buttonProps,
  closable = true,
  children,
}: ToasterProps) {
  const icon = (
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
        "border-primary bg-primary flex gap-2 rounded-xl border p-3 shadow-xl",
        {
          default: "text-primary",
          brand: "text-accent",
          success: "text-success",
          warning: "text-warning",
          error: "text-error",
        }[variant]
      )}
    >
      <div className="flex flex-1 gap-2">
        <Icon type={icon} size="md" />

        <div className="flex flex-col gap-1">
          <Typography as="div" weight="medium" className="leading-none">
            {title}
          </Typography>
          <Typography size="sm" className="text-secondary">
            {children}
          </Typography>
        </div>
      </div>

      {buttonProps && (
        <div className="flex items-center">
          <Button {...buttonProps} variant="tertiary" />
        </div>
      )}

      {closable && (
        <div className="flex items-center">
          <IconButton icon="close" variant="tertiary" slot="close" />
        </div>
      )}
    </div>
  );
}
