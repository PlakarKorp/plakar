import clsx from "clsx";
import { Meter } from "react-aria-components";

type ShareBarProps = {
  /** The share filled, from 0 to 100. */
  percent: number;
  label: string;
  /** The background class of the filled part, such as `bg-success`. */
  fillClassName?: string;
  className?: string;
};

/** A share of a whole, filled from the left on a muted track. */
export function ShareBar({
  percent,
  label,
  fillClassName = "bg-accent",
  className,
}: ShareBarProps) {
  const value = Math.min(100, Math.max(0, percent));

  return (
    <Meter
      value={value}
      aria-label={label}
      className={clsx(
        "bg-tertiary h-2 w-full overflow-hidden rounded-full",
        className
      )}
    >
      <div
        className={clsx("h-full rounded-full", fillClassName)}
        style={{ width: `${value}%` }}
      />
    </Meter>
  );
}
