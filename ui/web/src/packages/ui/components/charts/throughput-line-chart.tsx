import { useMemo } from "react";
import {
  Area,
  CartesianGrid,
  ComposedChart,
  Line,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

export type ThroughputPoint = {
  t: number;
  write: number;
  read: number;
  latency?: number;
};

type ThroughputLineChartProps = {
  data: ThroughputPoint[];
  formatBytes: (value: number) => string;
  formatLatency: (value: number) => string;
};

// Three distinct hues, not shades of one: the read and write areas are
// translucent and mirrored, so two tints of the same blue read as one shape.
// The -500 tokens hold their value in both themes, unlike the -secondary ones
// which darken in dark mode.
const WRITE_COLOR = "var(--background-color-accent)";
const READ_COLOR = "var(--background-color-success)";
const LATENCY_COLOR = "var(--background-color-warning)";

function formatElapsed(secs: number): string {
  if (secs < 60) return `${Math.round(secs)}s`;
  if (secs < 3600) return `${Math.round(secs / 60)}m`;
  return `${Math.round(secs / 360) / 10}h`;
}

// Mirrored throughput over time: writes above the zero line, reads below.
// Reads are negated to place them under the baseline, so every read value the
// axis, the tooltip or a reader sees must go back through Math.abs.
//
// Latency rides on its own right-hand axis: it is milliseconds against bytes
// per second, so the two cannot share a scale.
export function ThroughputLineChart({
  data,
  formatBytes,
  formatLatency,
}: ThroughputLineChartProps) {
  const maxT = data.length > 0 ? data[data.length - 1].t : 0;

  // One symmetric domain so a read and a write of equal size are equally tall;
  // an asymmetric axis would make the smaller direction look artificially calm.
  const bound = useMemo(() => {
    const peak = Math.max(
      0,
      ...data.map((d) => Math.max(Math.abs(d.write), Math.abs(d.read)))
    );
    return peak === 0 ? 1 : peak;
  }, [data]);

  const mirrored = useMemo(
    () => data.map((d) => ({ ...d, read: -Math.abs(d.read) })),
    [data]
  );

  const hasLatency = useMemo(
    () => data.some((d) => (d.latency ?? 0) > 0),
    [data]
  );

  return (
    <div className="flex h-full flex-col">
      <div className="min-h-0 flex-1">
        <ResponsiveContainer width="100%" height="100%">
          <ComposedChart
            margin={{ top: 4, right: 8, bottom: 0, left: 0 }}
            data={mirrored}
          >
            <Area
              yAxisId="bytes"
              type="monotone"
              dataKey="write"
              stroke={WRITE_COLOR}
              fill={WRITE_COLOR}
              fillOpacity={0.2}
              strokeWidth={2}
              dot={false}
              isAnimationActive={false}
              name="Write"
            />
            <Area
              yAxisId="bytes"
              type="monotone"
              dataKey="read"
              stroke={READ_COLOR}
              fill={READ_COLOR}
              fillOpacity={0.2}
              strokeWidth={2}
              dot={false}
              isAnimationActive={false}
              name="Read"
            />
            {hasLatency && (
              <Line
                yAxisId="latency"
                type="monotone"
                dataKey="latency"
                stroke={LATENCY_COLOR}
                strokeWidth={2}
                strokeDasharray="4 3"
                dot={false}
                isAnimationActive={false}
                name="Write p95"
                connectNulls
              />
            )}
            <ReferenceLine
              yAxisId="bytes"
              y={0}
              stroke="var(--text-color-quaternary)"
            />
            <XAxis
              dataKey="t"
              type="number"
              domain={[0, maxT]}
              tickFormatter={(v) => formatElapsed(Number(v))}
              tick={{ fill: "var(--text-color-quaternary)", fontSize: 12 }}
            />
            <YAxis
              yAxisId="bytes"
              orientation="left"
              domain={[-bound, bound]}
              tickFormatter={(v) =>
                Number(v) === 0 ? "" : formatBytes(Math.abs(Number(v)))
              }
              tick={{ fill: "var(--text-color-quaternary)", fontSize: 12 }}
            />
            {hasLatency && (
              <YAxis
                yAxisId="latency"
                orientation="right"
                tickFormatter={(v) =>
                  Number(v) === 0 ? "" : formatLatency(Number(v))
                }
                tick={{ fill: "var(--text-color-quaternary)", fontSize: 12 }}
              />
            )}
            <Tooltip
              wrapperClassName="!bg-primary rounded-lg shadow-lg border !border-secondary text-sm"
              labelFormatter={(v) => formatElapsed(Number(v))}
              formatter={(value, name) => {
                const n = Number(value);
                if (name === "Write p95") return formatLatency(n);
                return formatBytes(Math.abs(n));
              }}
            />
            <CartesianGrid
              stroke="var(--background-color-tertiary)"
              strokeDasharray="5 5"
            />
          </ComposedChart>
        </ResponsiveContainer>
      </div>

      <div className="text-quaternary flex shrink-0 items-center justify-center gap-4 pt-1 text-xs">
        <LegendKey color={WRITE_COLOR} label="Write" />
        <LegendKey color={READ_COLOR} label="Read" />
        {hasLatency && (
          <LegendKey color={LATENCY_COLOR} label="Write p95" dashed />
        )}
      </div>
    </div>
  );
}

function LegendKey({
  color,
  label,
  dashed,
}: {
  color: string;
  label: string;
  dashed?: boolean;
}) {
  return (
    <span className="flex items-center gap-1.5">
      <span
        className={
          dashed ? "h-0 w-3 border-t-2 border-dashed" : "h-2 w-2 rounded-full"
        }
        style={dashed ? { borderColor: color } : { backgroundColor: color }}
      />
      {label}
    </span>
  );
}
