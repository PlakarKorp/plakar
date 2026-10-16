import clsx from "clsx";
import { PropsWithChildren } from "react";
import { LinkProps, Link, Button as RAButton } from "react-aria-components";

import { Icon, IconSize, IconType, Typography, Spinner, Tooltip } from "..";
import { ButtonVariant, ButtonSize } from "./button.constants";

type CustomProps = Pick<LinkProps, "href" | "target"> &
  PropsWithChildren<{
    variant?: ButtonVariant;
    size?: ButtonSize;
    leftIcon?: IconType;
    rightIcon?: IconType;
    tooltip?: string;
    download?: boolean;
    isLoading?: boolean;
  }>;

type HeadlessButtonWithoutCustomProps = Omit<
  React.ComponentProps<typeof RAButton>,
  keyof CustomProps
>;

function getSizeClasses(size: ButtonSize, hasChildren: boolean) {
  return hasChildren
    ? {
        xs: "rounded px-1.5 py-0.5 gap-0.5",
        sm: "rounded-md px-2 py-0.5 gap-1",
        md: "rounded-lg px-3 py-2 gap-2",
      }[size]
    : // Icon button has no children and different padding
      {
        xs: "rounded p-0.5",
        sm: "rounded-md p-1",
        md: "rounded-lg p-2.5",
      }[size];
}

function getVariantClasses(variant: ButtonVariant) {
  return {
    primary:
      "bg-accent text-on-brand border border-accent data-hovered:bg-accent-hover data-hovered:border-background-accent-hover outline-offset-3",
    secondary:
      "border border-primary bg-primary text-secondary data-hovered:bg-primary-hover",
    tertiary:
      "text-tertiary data-hovered:text-tertiary-hover data-hovered:bg-primary-hover",
    error:
      "bg-error text-on-brand data-hovered:bg-error-secondary outline-offset-3",
    "error-tertiary":
      "text-tertiary data-hovered:text-error data-hovered:bg-error-tertiary",
    "accent-tertiary":
      "bg-accent-tertiary text-accent data-hovered:bg-accent-tertiary-hover",
    "accent-quaternary": "text-accent data-hovered:bg-accent-tertiary",
    "warning-quaternary": "text-warning data-hovered:bg-warning-tertiary",
    "error-quaternary": "text-error data-hovered:bg-error-tertiary",
    "success-quaternary": "text-success data-hovered:bg-success-tertiary",
  }[variant];
}

export type ButtonProps = CustomProps & HeadlessButtonWithoutCustomProps;

export function Button({
  variant = "primary",
  size = "md",
  leftIcon,
  rightIcon,
  children,
  tooltip,
  href,
  download,
  target,
  isLoading,
  ...props
}: ButtonProps) {
  const showSpinner = isLoading || props.isPending;
  const className = clsx(
    "relative inline-flex shrink-0 cursor-pointer items-center justify-center",
    "data-disabled:border-background-disable data-disabled:bg-disable data-disabled:text-disable data-disabled:cursor-default",
    "data-focused:outline-focus",
    getSizeClasses(size, !!children),
    getVariantClasses(variant),
    props.className
  );

  const content = (
    <>
      {leftIcon && (
        <Icon
          type={leftIcon}
          size={size}
          className={clsx(showSpinner && "invisible")}
        />
      )}
      {children && (
        <Typography
          as="span"
          size={size === "xs" ? "xs" : "sm"}
          weight="medium"
          className={clsx(showSpinner && "invisible")}
        >
          {children}
        </Typography>
      )}
      {rightIcon && (
        <Icon
          type={rightIcon}
          size={size}
          className={clsx(showSpinner && "invisible")}
        />
      )}
      {showSpinner && (
        <Spinner
          className="absolute"
          size={
            {
              xs: "sm",
              sm: "md",
              md: "lg",
            }[size] as IconSize
          }
          inheritColor
        />
      )}
    </>
  );

  const buttonElement = href ? (
    <Link
      href={href}
      className={className}
      download={download}
      target={target}
      rel={target === "_blank" ? "noopener noreferrer" : undefined}
      aria-label={props["aria-label"]}
      isDisabled={props.isDisabled}
    >
      {content}
    </Link>
  ) : (
    <RAButton {...props} className={className}>
      {content}
    </RAButton>
  );

  return tooltip && !props.isDisabled ? (
    <Tooltip label={tooltip}>{buttonElement}</Tooltip>
  ) : (
    buttonElement
  );
}
