import { Typography } from "@plakar-ui/ui/components";
import clsx from "clsx";

type DeltaProps = {
  before: number;
  after: number;
  format: (value: number) => string;
  // Whether a smaller value is the good direction, as for errors.
  lowerIsBetter?: boolean;
  className?: string;
};

/**
 * Shows how a value moved between two restore points, signed and coloured
 * by whether the move is good.
 */
export function Delta({
  before,
  after,
  format,
  lowerIsBetter,
  className,
}: DeltaProps) {
  const delta = after - before;
  const isGood = lowerIsBetter ? delta < 0 : delta > 0;
  return (
    <Typography
      as="span"
      size="sm"
      className={clsx(
        "tabular-nums",
        delta === 0 ? "text-tertiary" : isGood ? "text-success" : "text-error",
        className
      )}
    >
      {delta === 0 ? "=" : `${delta > 0 ? "+" : "−"}${format(Math.abs(delta))}`}
    </Typography>
  );
}
