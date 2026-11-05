import clsx from "clsx";
import { ReactNode } from "react";

import { Typography } from "..";
import { CopyButton } from "./copy-button";
import {
  TypographySize,
  TypographyWeight,
} from "../typography/typography.constants";

const COPY_BUTTON_VISIBILITY = {
  always: undefined,
  // Where a pointer can hover, the button waits for the closest `group/row`
  // ancestor to be hovered, or for the keyboard to reach it.
  "on-row-hover":
    "[@media(hover:hover)]:opacity-0 group-hover/row:opacity-100 data-focus-visible:opacity-100",
} as const;

type IdProps = {
  wrap?: (children: ReactNode) => ReactNode;
  value: string;
  helpText?: string;
  size?: TypographySize;
  weight?: TypographyWeight;
  withCopyButton?: boolean;
  /** Shows the copy button always, or only while the row holding the ID is hovered. */
  copyButtonVisibility?: keyof typeof COPY_BUTTON_VISIBILITY;
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
  copyButtonVisibility = "always",
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
        <CopyButton
          content={value}
          tooltip={helpText}
          className={clsx(
            "-my-1",
            COPY_BUTTON_VISIBILITY[copyButtonVisibility]
          )}
        />
      )}
    </span>
  );
}
