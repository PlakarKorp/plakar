import { ReactNode } from "react";
import { DialogTrigger, PopoverProps } from "react-aria-components";

import { Button, ButtonProps } from "./button";
import { DropdownListPopover } from "../dropdown-list/dropdown-list";

type ButtonDropdownProps = Omit<
  ButtonProps,
  "children" | "rightIcon" | "tooltip" | "onPress"
> &
  Pick<PopoverProps, "placement"> & {
    label: string;
    children: (close: () => void) => ReactNode;
  };

export function ButtonDropdown({
  label,
  variant = "secondary",
  placement = "bottom end",
  children,
  ...props
}: ButtonDropdownProps) {
  return (
    <DialogTrigger>
      <Button {...props} variant={variant} rightIcon="chevron-down">
        {label}
      </Button>
      <DropdownListPopover label={label} placement={placement}>
        {children}
      </DropdownListPopover>
    </DialogTrigger>
  );
}
