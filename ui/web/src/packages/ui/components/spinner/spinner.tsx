import clsx from "clsx";
import { ProgressBar } from "react-aria-components";

import { Icon, IconSize } from "..";

type SpinnerProps = {
  size?: IconSize;
  inheritColor?: boolean;
  className?: string;
};

export function Spinner({
  size = "6xl",
  inheritColor = false,
  className,
}: SpinnerProps) {
  return (
    <ProgressBar
      aria-label="Loading…"
      value={80}
      className={clsx("flex h-full items-center justify-center", className)}
      isIndeterminate
    >
      <Icon
        type="spinner"
        size={size}
        className={clsx(
          "animate-spin",
          inheritColor ? "text-current" : "text-accent-hover"
        )}
      />
    </ProgressBar>
  );
}
