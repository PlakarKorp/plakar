import clsx from "clsx";
import { ReactNode } from "react";

import { Typography } from "../typography/typography";

type StatCountersProps = {
  items: { label: string; value: ReactNode }[];
  className?: string;
};

/** A row of figures, each under its label, split by thin rules. */
export function StatCounters({ items, className }: StatCountersProps) {
  return (
    <div
      className={clsx(
        "divide-secondary grid auto-cols-fr grid-flow-col divide-x",
        className
      )}
    >
      {items.map((item) => (
        <div
          key={item.label}
          className="flex min-w-0 flex-col justify-center gap-0.5 px-3 first:pl-0 last:pr-0"
        >
          <Typography size="xs" className="text-tertiary truncate">
            {item.label}
          </Typography>
          <Typography
            as="p"
            variant="heading"
            size="sm"
            weight="medium"
            className="text-primary truncate tabular-nums"
          >
            {item.value}
          </Typography>
        </div>
      ))}
    </div>
  );
}
