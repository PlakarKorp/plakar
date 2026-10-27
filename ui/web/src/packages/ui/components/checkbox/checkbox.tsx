import clsx from "clsx";
import { ReactNode } from "react";
import {
  Checkbox as RACCheckbox,
  CheckboxProps as RACCheckboxProps,
} from "react-aria-components";

import { Typography, Icon } from "..";
import { CheckboxSizes } from "./checkbox.constants";

export type CheckboxProps = RACCheckboxProps & {
  size?: (typeof CheckboxSizes)[number];
  label?: ReactNode;
  description?: string;
};

export function Checkbox({
  size = "md",
  label,
  description,
  ...props
}: CheckboxProps) {
  return (
    <RACCheckbox
      {...props}
      className={clsx(
        props.className,
        "group/cb relative inline-flex items-baseline gap-2"
      )}
    >
      {({ isSelected, isIndeterminate, isDisabled, isFocusVisible }) => {
        const isActive = isSelected || isIndeterminate;

        return (
          <>
            <span className="relative inline-flex">
              <div
                className={clsx(
                  "relative flex items-center justify-center border transition-colors",
                  size === "md" ? "rounded-md p-0.5" : "rounded-sm p-0.25",

                  isActive
                    ? isDisabled
                      ? "border-background-accent-tertiary bg-accent-tertiary"
                      : "border-accent bg-accent group-hover/cb:border-background-accent-hover group-hover/cb:bg-accent-hover"
                    : isDisabled
                      ? "border-background-disable bg-disable"
                      : "border-primary bg-primary group-hover/cb:bg-primary-hover",

                  isFocusVisible && "outline-focus outline-2 outline-offset-2"
                )}
              >
                {isIndeterminate ? (
                  <Icon
                    type="minus"
                    size={size === "md" ? "sm" : "xs"}
                    className="text-on-brand"
                    aria-hidden="true"
                  />
                ) : (
                  <Icon
                    type="check"
                    size={size === "md" ? "sm" : "xs"}
                    className={clsx(
                      "text-on-brand",
                      isSelected ? "visible" : "invisible"
                    )}
                    aria-hidden="true"
                  />
                )}
              </div>

              <span
                className="absolute inset-0 -m-2 rounded-full"
                aria-hidden="true"
              />
            </span>

            {label && (
              <span className="flex flex-col gap-1">
                <Typography
                  as="span"
                  variant="body"
                  weight="medium"
                  className="text-secondary group-disabled/cb:text-disable"
                  size={size === "md" ? "md" : "sm"}
                >
                  {label}
                </Typography>

                {description && (
                  <Typography
                    as="span"
                    variant="body"
                    className="text-tertiary group-disabled/cb:text-disable"
                    size={size === "md" ? "sm" : "xs"}
                  >
                    {description}
                  </Typography>
                )}
              </span>
            )}
          </>
        );
      }}
    </RACCheckbox>
  );
}
