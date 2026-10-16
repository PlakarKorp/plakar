import { ReactNode } from "react";
import {
  Tooltip,
  ResponsiveContainer,
  PieChart as RechartsPieChart,
  Pie,
  PieSectorShapeProps,
  Sector,
} from "recharts";

type PieChartProps = {
  data: {
    name: string;
    value: number;
    color: string;
  }[];
  width?: number | `${number}%`;
  height?: number | `${number}%`;
  label?: ReactNode;
};

export function PieChart({
  data,
  width = "100%",
  height = "100%",
  label,
}: PieChartProps) {
  return (
    <ResponsiveContainer width={width} height={height}>
      <RechartsPieChart responsive>
        <Pie
          data={data}
          labelLine={false}
          dataKey="value"
          shape={MyCustomPie}
          innerRadius="70%"
          outerRadius="100%"
          paddingAngle={2}
        />
        {label !== undefined && (
          <>
            {typeof label === "string" || typeof label === "number" ? (
              <text
                x="50%"
                y="50%"
                textAnchor="middle"
                dominantBaseline="central"
                style={{
                  fontSize: 14,
                  fill: "currentColor",
                  pointerEvents: "none",
                }}
              >
                {label}
              </text>
            ) : (
              <foreignObject
                x="50%"
                y="50%"
                width={48}
                height={48}
                transform="translate(-24,-24)"
                style={{
                  overflow: "visible",
                  pointerEvents: "none",
                }}
              >
                <div
                  style={{
                    width: "100%",
                    height: "100%",
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                  }}
                >
                  {label}
                </div>
              </foreignObject>
            )}
          </>
        )}
        <Tooltip wrapperClassName="!bg-primary rounded-lg shadow-lg border !border-secondary text-sm" />
      </RechartsPieChart>
    </ResponsiveContainer>
  );
}

const MyCustomPie = (props: PieSectorShapeProps) => {
  return <Sector {...props} fill={props.color} stroke="none" strokeWidth={0} />;
};
