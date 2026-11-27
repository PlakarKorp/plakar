import clsx from "clsx";

import { Tooltip } from "../tooltip/tooltip";

export type Segment = {
  label: string;
  value: number;
  /** The background class that colors the segment, such as `bg-success`. */
  className: string;
};

type SegmentBarProps = {
  segments: Segment[];
  className?: string;
};

/** Shares of a whole, side by side, each named in its tooltip. */
export function SegmentBar({ segments, className }: SegmentBarProps) {
  const shown = segments.filter((segment) => segment.value > 0);

  return (
    <div
      className={clsx(
        "flex h-2 w-full gap-0.5",
        shown.length === 0 && "bg-tertiary rounded-full",
        className
      )}
    >
      {shown.map((segment) => {
        const label = `${segment.label}: ${segment.value.toLocaleString("en-US")}`;

        return (
          <Tooltip key={segment.label} label={label}>
            <span
              role="img"
              aria-label={label}
              className={clsx("h-full rounded-full", segment.className)}
              style={{ flexGrow: segment.value }}
            />
          </Tooltip>
        );
      })}
    </div>
  );
}
