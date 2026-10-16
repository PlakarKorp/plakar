import clsx from "clsx";
import * as React from "react";

import { IconButton } from "../icon-button/icon-button";
import { Typography } from "../typography/typography";

type NoticeBannerVariant = "info" | "danger";

type NoticeBannerProps = {
  children: React.ReactNode;
  variant?: NoticeBannerVariant;
  dismissible?: boolean;
  defaultOpen?: boolean;
  onDismiss?: () => void;
  onOpenChange?: (isOpen: boolean) => void;
  className?: string;
  "aria-label"?: string;
};

export function NoticeBanner({
  children,
  variant = "info",
  dismissible = true,
  defaultOpen = true,
  onDismiss,
  onOpenChange,
  className,
  "aria-label": ariaLabel,
}: NoticeBannerProps) {
  const [isOpen, setIsOpen] = React.useState(defaultOpen);

  const close = React.useCallback(() => {
    setIsOpen(false);
    onOpenChange?.(false);
    onDismiss?.();
  }, [onDismiss, onOpenChange]);

  if (!isOpen) return null;

  return (
    <section
      role={variant === "danger" ? "alert" : "status"}
      aria-label={ariaLabel}
      className={clsx(
        "flex w-full items-center gap-4 px-4 shadow-xl",
        {
          info: "bg-accent-tertiary text-accent",
          danger: "bg-error text-on-brand",
        }[variant],
        className
      )}
    >
      <div className="mx-auto flex max-w-7xl items-start justify-between gap-3 py-2">
        <Typography as="div" size="sm" className="min-w-0 flex-1">
          {children}
        </Typography>
      </div>

      {dismissible ? (
        <IconButton
          variant={variant === "danger" ? "error" : "accent-tertiary"}
          onPress={close}
          aria-label="Dismiss notice"
          icon="close"
          size="sm"
          className="shrink-0"
        />
      ) : null}
    </section>
  );
}
