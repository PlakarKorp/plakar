import clsx from "clsx";
import { PropsWithChildren, ReactNode } from "react";
import { Button } from "react-aria-components";

import { Tooltip } from "..";

type HintProps = PropsWithChildren & {
  tooltip: ReactNode;
  underline?: "always" | "hover";
  color?: "inherit" | "accent";
};

export function Hint({
  tooltip,
  color = "accent",
  underline = "always",
  children,
}: HintProps) {
  return (
    <Tooltip label={tooltip} offset={2} maxWidth={410}>
      <Button
        className={clsx(
          "cursor-help decoration-dashed decoration-1 underline-offset-2 hover:underline",
          color === "accent" && "text-accent hover:text-accent-hover",
          color === "inherit" && "hover:opacity-80",
          underline === "always" && "underline"
        )}
      >
        {children}
      </Button>
    </Tooltip>
  );
}
