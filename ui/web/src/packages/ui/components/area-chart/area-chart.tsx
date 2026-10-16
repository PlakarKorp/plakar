type DataPoint = {
  value: number;
};

type Point = {
  x: number;
  y: number;
};

const getX = (
  index: number,
  total: number,
  width: number,
  padding: number
): number => {
  const step = (width - padding * 2) / (total - 1);
  return padding + index * step;
};

const getY = (
  value: number,
  maxValue: number,
  height: number,
  padding: number,
  strokeBuffer: number
): number => {
  const drawableHeight = height - padding * 2 - strokeBuffer;

  // If all values are zero, we return the bottom of the chart
  if (maxValue === 0) {
    return height - padding - strokeBuffer;
  }

  const scale = drawableHeight / maxValue;
  const y = height - padding - value * scale;
  const yAdjusted = value === 0 ? y - strokeBuffer : y;

  return Math.max(yAdjusted, strokeBuffer);
};

const buildPath = (points: Point[]): string => {
  if (points.length < 2) return "";

  let d = `M ${points[0].x},${points[0].y}`;

  for (let i = 1; i < points.length; i++) {
    const p0 = points[i - 1];
    const p1 = points[i];
    const midX = (p0.x + p1.x) / 2;

    const cp1x = midX;
    const cp1y = p0.y;
    const cp2x = midX;
    const cp2y = p1.y;

    d += ` C ${cp1x},${cp1y} ${cp2x},${cp2y} ${p1.x},${p1.y}`;
  }

  return d;
};

type AreaChartProps = {
  data: DataPoint[];
  width?: number;
  height?: number;
  strokeWidth?: number;
  padding?: number;
};

export function AreaChart({
  data = [],
  width = 100,
  height = 100,
  strokeWidth = 3,
  padding = 0,
}: AreaChartProps) {
  if (!data.length) return null;

  const strokeBuffer = strokeWidth / 2;
  const maxValue = Math.max(...data.map((d) => d.value));

  const points = data.map((d, i) => ({
    x: getX(i, data.length, width, padding),
    y: getY(d.value, maxValue, height, padding, strokeBuffer),
  }));

  const pathD = buildPath(points);
  const areaD = `${pathD} L ${points[points.length - 1].x},${height - padding} L ${points[0].x},${height - padding} Z`;

  return (
    <svg width={width} height={height}>
      <path d={areaD} className="fill-background-accent-tertiary" />
      <path
        d={pathD}
        fill="none"
        className="stroke-border-accent"
        strokeWidth={strokeWidth}
      />
    </svg>
  );
}
