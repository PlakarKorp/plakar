import {
  CartesianGrid,
  Bar,
  BarChart,
  Tooltip,
  XAxis,
  YAxis,
  ResponsiveContainer,
} from "recharts";

type JobsChartProps = {
  data: {
    name: string;
    completed: number;
    failed: number;
  }[];
};

export function JobsBarChart({ data }: JobsChartProps) {
  return (
    <ResponsiveContainer width="100%" height="100%">
      <BarChart
        margin={{
          top: 1,
          right: 24,
          bottom: 0,
          left: -20,
        }}
        data={data}
      >
        <Bar
          dataKey="completed"
          fill="var(--background-color-success)"
          name="Completed"
        />
        <Bar
          dataKey="failed"
          fill="var(--background-color-error)"
          name="Failed"
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
        <Tooltip wrapperClassName="!bg-primary rounded-lg shadow-lg border !border-secondary text-sm" />
        <CartesianGrid
          stroke="var(--background-color-tertiary)"
          strokeDasharray="5 5"
          vertical={false}
        />
      </BarChart>
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
  return (
    <text
      x={x}
      y={y}
      textAnchor="end"
      fill="var(--text-color-quaternary)"
      fontSize={12}
    >
      {payload.value}
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
