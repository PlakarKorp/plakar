import clsx from "clsx";
import { PropsWithChildren } from "react";

import { Typography } from "../typography/typography";

type FieldSupportTextProps = PropsWithChildren & {
  className?: string;
};

export function FieldSupportText({
  children,
  className,
}: FieldSupportTextProps) {
  return (
    <Typography
      variant="body"
      size="xs"
      className={clsx(
        "text-quaternary",
        "group-data-invalid:text-error",
        className
      )}
    >
      {children}
    </Typography>
  );
}
