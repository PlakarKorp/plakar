import { Typography } from "..";

type PieProps = {
  value?: number;
  size?: number;
  strokeWidth?: number;
};

export function Pie({ value = 0, size = 100, strokeWidth = 10 }: PieProps) {
  const clampedValue = Math.max(0, Math.min(100, value));
  const radius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;
  const offset = circumference * (1 - (clampedValue * 0.97 + 1) / 100);

  const center = size / 2;

  const circleProps = {
    cx: center,
    cy: center,
    r: radius,
    strokeWidth,
    fill: "transparent",
  };

  return (
    <div className="relative" style={{ width: size, height: size }}>
      <svg width={size} height={size}>
        <circle {...circleProps} className="stroke-border-secondary" />

        {clampedValue > 0 && clampedValue < 100 && (
          <>
            <circle
              {...circleProps}
              className="stroke-border-accent"
              strokeDasharray={circumference}
              strokeDashoffset={offset}
              strokeLinecap="butt"
              transform={`rotate(-90 ${center} ${center})`}
            />
            <circle
              {...circleProps}
              className="stroke-border-accent"
              strokeDasharray={`0 ${circumference}`}
              strokeDashoffset={offset}
              strokeLinecap="round"
              transform={`rotate(-90 ${center} ${center})`}
            />
          </>
        )}

        {clampedValue === 100 && (
          <circle
            {...circleProps}
            className="stroke-border-accent"
            strokeDasharray={360}
          />
        )}
      </svg>

      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2">
        <Typography variant="body" size="md" className="text-primary" monospace>
          {Math.round(clampedValue)}%
        </Typography>
      </div>
    </div>
  );
}
