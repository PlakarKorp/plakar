import clsx from "clsx";

import { Typography } from "../typography/typography";

export type ChartLegendItem = {
  label: string;
  /** The classes that draw the mark of the series, its color and shape. */
  markClassName: string;
};

type ChartLegendProps = {
  items: ChartLegendItem[];
  className?: string;
};

/** The key to the colors of a chart. */
export function ChartLegend({ items, className }: ChartLegendProps) {
  if (items.length === 0) {
    return null;
  }

  return (
    <div className={clsx("flex gap-4", className)}>
      {items.map((item) => (
        <Typography
          key={item.label}
          as="div"
          size="xs"
          className="flex items-center gap-1.5 whitespace-nowrap"
        >
          <div className={item.markClassName} />
          <span className="text-tertiary">{item.label}</span>
        </Typography>
      ))}
    </div>
  );
}
