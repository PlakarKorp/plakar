import clsx from "clsx";
import { ReactNode } from "react";
import {
  ToggleButton,
  ToggleButtonGroup as AriaToggleButtonGroup,
  ToggleButtonGroupProps as AriaToggleButtonGroupProps,
  ToggleButtonProps,
} from "react-aria-components";

import { Icon, IconType, Typography } from "..";
import { FieldLabel } from "../field/field-label";
import { FieldSupportText } from "../field/field-support-text";

export type TagToggleGroupRootProps = Omit<
  AriaToggleButtonGroupProps,
  "children"
> & {
  children?: ReactNode;
  label?: string;
  supportText?: string;
  isRequired?: boolean;
  isInvalid?: boolean;
};

export function TagToggleGroupRoot({
  label,
  supportText,
  className,
  children,
  isRequired = false,
  isInvalid = false,
  ...props
}: TagToggleGroupRootProps) {
  return (
    <AriaToggleButtonGroup
      {...props}
      data-invalid={isInvalid || undefined}
      className={clsx("group flex flex-col gap-2", className)}
    >
      {label && <FieldLabel isRequired={isRequired}>{label}</FieldLabel>}
      <div className="flex flex-wrap gap-2">{children}</div>
      {supportText && <FieldSupportText>{supportText}</FieldSupportText>}
    </AriaToggleButtonGroup>
  );
}

type TagToggleGroupItemProps = Omit<ToggleButtonProps, "children"> & {
  children?: ReactNode;
  leftIcon?: IconType;
  rightIcon?: IconType;
};

export function TagToggleGroupItem({
  leftIcon,
  rightIcon,
  children,
  className,
  ...props
}: TagToggleGroupItemProps) {
  return (
    <ToggleButton
      {...props}
      className={clsx(
        "inline-flex cursor-pointer items-center gap-1.5",
        "rounded-2xl border px-3 py-1",
        // Default state
        "border-secondary bg-primary text-secondary",
        // Hover
        "data-hovered:bg-primary-hover data-hovered:text-primary",
        // Selected
        "data-selected:border-accent-heavy data-selected:bg-accent-heavy data-selected:text-on-accent data-selected:data-hovered:bg-accent-heavy data-selected:data-hovered:text-on-accent",
        // Focus visible
        "data-focus-visible:outline-focus",
        // Disabled
        "data-disabled:border-background-disable data-disabled:bg-disable data-disabled:text-disable data-disabled:cursor-not-allowed",
        "duration-200 motion-safe:transition-colors",
        className
      )}
    >
      {leftIcon && <Icon type={leftIcon} size="sm" />}
      <Typography as="span" size="sm">
        {children}
      </Typography>
      {rightIcon && <Icon type={rightIcon} size="sm" />}
    </ToggleButton>
  );
}

export const TagToggleGroup = Object.assign(TagToggleGroupRoot, {
  Button: TagToggleGroupItem,
});
