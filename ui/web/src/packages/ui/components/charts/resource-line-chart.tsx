import { useMemo } from "react";
import {
  CartesianGrid,
  Line,
  LineChart,
  Tooltip,
  XAxis,
  YAxis,
  ResponsiveContainer,
} from "recharts";

export type SeriesDef<K extends string = string> = {
  key: K;
  name: string;
  format: (value: number) => string;
};

type ResourceLineChartProps<K extends string> = {
  data: Array<{ t: number } & Record<K, number>>;
  series: [SeriesDef<K>] | [SeriesDef<K>, SeriesDef<K>];
};

// Distinct hues rather than two tints of the brand blue, which read as one
// series on a small chart. The -500 tokens hold their value in both themes.
const SERIES_COLORS = [
  "var(--background-color-accent)",
  "var(--background-color-success)",
] as const;

function formatElapsed(secs: number): string {
  if (secs < 60) return `${Math.round(secs)}s`;
  if (secs < 3600) return `${Math.round(secs / 60)}m`;
  return `${Math.round(secs / 360) / 10}h`;
}

type TickProps = {
  x: string | number;
  y: string | number;
  payload: { value: number };
};

function makeYAxisTick(format: (v: number) => string, anchor: "end" | "start") {
  return ({ x, y, payload }: TickProps) => (
    <text
      x={x}
      y={y}
      textAnchor={anchor}
      fill="var(--text-color-quaternary)"
      fontSize={12}
    >
      {payload.value === 0 ? "" : format(payload.value)}
    </text>
  );
}

// Line chart for time-series data with up to two metrics on separate Y axes.
// The X axis is real elapsed time (numeric), so decimated/non-uniform samples
// land at their true position.
export function ResourceLineChart<K extends string>({
  data,
  series,
}: ResourceLineChartProps<K>) {
  const maxT = data.length > 0 ? data[data.length - 1].t : 0;
  const s0 = series[0];
  const s1 = series.length > 1 ? series[1] : undefined;

  const leftTick = useMemo(() => makeYAxisTick(s0.format, "end"), [s0.format]);
  const rightTick = useMemo(
    () => (s1 ? makeYAxisTick(s1.format, "start") : undefined),
    [s1]
  );

  return (
    <div className="flex h-full flex-col">
      <div className="min-h-0 flex-1">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart
            margin={{ top: 4, right: 8, bottom: 0, left: 0 }}
            data={data}
          >
            <Line
              yAxisId="left"
              type="monotone"
              dataKey={s0.key}
              stroke={SERIES_COLORS[0]}
              strokeWidth={2}
              dot={false}
              isAnimationActive={false}
              name={s0.name}
            />
            {s1 && (
              <Line
                yAxisId="right"
                type="monotone"
                dataKey={s1.key}
                stroke={SERIES_COLORS[1]}
                strokeWidth={2}
                dot={false}
                isAnimationActive={false}
                name={s1.name}
              />
            )}
            <XAxis
              dataKey="t"
              type="number"
              domain={[0, maxT]}
              tickFormatter={(v) => formatElapsed(Number(v))}
              tick={{ fill: "var(--text-color-quaternary)", fontSize: 12 }}
            />
            <YAxis yAxisId="left" orientation="left" tick={leftTick} />
            {s1 && (
              <YAxis yAxisId="right" orientation="right" tick={rightTick} />
            )}
            <Tooltip
              wrapperClassName="!bg-primary rounded-lg shadow-lg border !border-secondary text-sm"
              labelFormatter={(v) => formatElapsed(Number(v))}
              formatter={(value, _name, item) => {
                const s = series.find((s) => s.key === item.dataKey);
                return typeof value === "number" && s
                  ? s.format(value)
                  : String(value);
              }}
            />
            <CartesianGrid
              stroke="var(--background-color-tertiary)"
              strokeDasharray="5 5"
            />
          </LineChart>
        </ResponsiveContainer>
      </div>

      <div className="text-quaternary flex shrink-0 items-center justify-center gap-4 pt-1 text-xs">
        <span className="flex items-center gap-1.5">
          <span
            className="h-2 w-2 rounded-full"
            style={{ backgroundColor: SERIES_COLORS[0] }}
          />
          {s0.name}
        </span>
        {s1 && (
          <span className="flex items-center gap-1.5">
            <span
              className="h-2 w-2 rounded-full"
              style={{ backgroundColor: SERIES_COLORS[1] }}
            />
            {s1.name}
          </span>
        )}
      </div>
    </div>
  );
}
