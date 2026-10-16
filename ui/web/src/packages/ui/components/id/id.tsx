import clsx from "clsx";
import { ReactNode } from "react";

import { Typography } from "..";
import { CopyButton } from "./copy-button";
import {
  TypographySize,
  TypographyWeight,
} from "../typography/typography.constants";

type IdProps = {
  wrap?: (children: ReactNode) => ReactNode;
  value: string;
  helpText?: string;
  size?: TypographySize;
  weight?: TypographyWeight;
  withCopyButton?: boolean;
  className?: string;
  noTruncate?: boolean;
};

export function Id({
  wrap = (children) => children,
  value,
  helpText = "Copy full ID",
  size,
  weight,
  withCopyButton = true,
  className,
  noTruncate = false,
}: IdProps) {
  const shortValue = noTruncate ? value : value.slice(0, 8);

  return (
    <span className={clsx("flex items-center gap-1 align-middle", className)}>
      <Typography as="span" size={size} weight={weight} monospace>
        {wrap(shortValue)}
      </Typography>

      {withCopyButton && (
        <CopyButton content={value} tooltip={helpText} className="-my-1" />
      )}
    </span>
  );
}
