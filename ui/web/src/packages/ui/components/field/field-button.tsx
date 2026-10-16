import clsx from "clsx";
import { Button, ButtonProps } from "react-aria-components";

import { Icon } from "../icon/icon";
import { IconType } from "../icon/icon.constants";

type FieldButtonProps = ButtonProps & {
  icon: IconType;
};

export function FieldButton({ className, icon, ...props }: FieldButtonProps) {
  return (
    <Button
      {...props}
      className={clsx(
        "outline-focus flex h-8.5 w-8.5 items-center justify-center rounded-full",
        "text-tertiary data-hovered:text-secondary data-disabled:text-disable",
        className
      )}
    >
      <Icon type={icon} size="md" />
    </Button>
  );
}
