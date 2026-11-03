import clsx from "clsx";

import { Tooltip } from "../tooltip/tooltip";

type SparkBarsProps = {
  data: { label: string; value: number }[];
  className?: string;
};

export function SparkBars({ data, className }: SparkBarsProps) {
  const max = Math.max(1, ...data.map((point) => point.value));

  return (
    <div className={clsx("flex h-9 items-end gap-0.5", className)}>
      {data.map((point) => (
        <Tooltip key={point.label} label={`${point.label}: ${point.value}`}>
          <span
            role="img"
            aria-label={`${point.label}: ${point.value}`}
            className={clsx(
              "flex-1 rounded-[1px]",
              point.value > 0 ? "bg-accent" : "bg-border-secondary"
            )}
            style={{
              height:
                point.value > 0
                  ? `${Math.max(18, (point.value / max) * 100)}%`
                  : "3px",
            }}
          />
        </Tooltip>
      ))}
    </div>
  );
}
