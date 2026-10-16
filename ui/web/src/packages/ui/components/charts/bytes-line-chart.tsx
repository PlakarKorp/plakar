import prettyBytes from "pretty-bytes";
import {
  CartesianGrid,
  Area,
  AreaChart,
  Tooltip,
  XAxis,
  YAxis,
  ResponsiveContainer,
} from "recharts";

type BytesChartProps = {
  data: {
    name: string;
    processed: number;
    stored: number;
  }[];
};

export function BytesLineChart({ data }: BytesChartProps) {
  return (
    <ResponsiveContainer width="100%" height="100%">
      <AreaChart
        margin={{
          top: 1,
          right: 24,
          bottom: 0,
          left: 0,
        }}
        data={data}
      >
        <Area
          type="monotone"
          dataKey="processed"
          stroke="var(--background-color-accent-secondary)"
          strokeWidth={2}
          fill="var(--background-color-accent-secondary)"
          name="Bytes processed"
        />
        <Area
          type="monotone"
          dataKey="stored"
          stroke="var(--background-color-accent)"
          strokeWidth={2}
          fill="var(--background-color-accent)"
          name="Bytes stored"
        />
        <XAxis
          dataKey="name"
          tick={({ x, y, payload }) => (
            <CustomXTick x={x} y={y} payload={payload} />
          )}
        />
        <YAxis
          tick={({ x, y, payload }) => (
            <CustomYTick x={x} y={y} payload={payload} />
          )}
          niceTicks="snap125"
        />
        <Tooltip
          wrapperClassName="!bg-primary rounded-lg shadow-lg border !border-secondary text-sm"
          formatter={(value) =>
            typeof value === "number" ? prettyBytes(value) : String(value)
          }
        />
        <CartesianGrid
          stroke="var(--background-color-tertiary)"
          strokeDasharray="5 5"
        />
      </AreaChart>
    </ResponsiveContainer>
  );
}

type CustomTickProps = {
  x: string | number;
  y: string | number;
  payload: {
    value: number;
  };
};

function CustomYTick({ x, y, payload }: CustomTickProps) {
  const label = payload.value === 0 ? "" : prettyBytes(payload.value);

  return (
    <text
      x={x}
      y={y}
      textAnchor="end"
      fill="var(--text-color-quaternary)"
      fontSize={12}
    >
      {label}
    </text>
  );
}

function CustomXTick({ x, y, payload }: CustomTickProps) {
  return (
    <text
      x={x}
      y={Number(y) + 12}
      textAnchor="middle"
      fill="var(--text-color-quaternary)"
      fontSize={12}
    >
      {payload.value}
    </text>
  );
}
