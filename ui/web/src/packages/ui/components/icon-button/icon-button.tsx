import { ComponentProps } from "react";

import { Button, IconType } from "..";

type IconButtonProps = Omit<
  ComponentProps<typeof Button>,
  "label" | "leftIcon" | "rightIcon"
> & {
  icon: IconType;
};

export function IconButton({ icon, ...props }: IconButtonProps) {
  return <Button leftIcon={icon} {...props} />;
}
