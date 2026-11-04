import clsx from "clsx";

import { Tooltip } from "../tooltip/tooltip";

type SparkBarsProps = {
  data: {
    label: string;
    value: number;
    tone?: "accent" | "success" | "warning" | "error";
    tooltip?: string;
  }[];
  className?: string;
};

export function SparkBars({ data, className }: SparkBarsProps) {
  const max = Math.max(1, ...data.map((point) => point.value));

  return (
    <div className={clsx("flex h-9 items-end gap-0.5", className)}>
      {data.map((point, index) => {
        const tooltip = point.tooltip ?? `${point.label}: ${point.value}`;

        return (
          <Tooltip key={point.label + index} label={tooltip}>
            <span
              role="img"
              aria-label={tooltip}
              className={clsx(
                "flex-1 rounded-[1px]",
                point.value > 0
                  ? {
                      accent: "bg-accent",
                      success: "bg-success",
                      warning: "bg-warning",
                      error: "bg-error",
                    }[point.tone ?? "accent"]
                  : "bg-border-secondary"
              )}
              style={{
                height:
                  point.value > 0
                    ? `${Math.max(18, (point.value / max) * 100)}%`
                    : "3px",
              }}
            />
          </Tooltip>
        );
      })}
    </div>
  );
}
