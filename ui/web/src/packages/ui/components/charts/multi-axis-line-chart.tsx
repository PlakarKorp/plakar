import { useEffect, useRef, useState } from "react";

import { LineChartAxis, LineChartLine } from "./multi-axis-line-chart.utils";

type MultiAxisLineChartProps = {
  times: number[];
  lines: LineChartLine[];
  left: LineChartAxis;
  right?: LineChartAxis;
  label: string;
  height?: number;
  // Names a time, on the axis and in the tooltip.
  formatTime?: (t: number) => string;
  // Times the axis labels; five even steps when unset.
  timeTicks?: number[];
};

const MARGIN_TOP = 8;
const MARGIN_BOTTOM = 24;
const MARGIN_SIDE = 4;
const CHAR_WIDTH = 7.4;
const X_TICKS = 5;

function formatElapsed(secs: number): string {
  if (secs < 60) return `${Math.round(secs)}s`;
  if (secs < 3600) return `${Math.round(secs / 60)}m`;
  return `${Math.round(secs / 360) / 10}h`;
}

function useWidth() {
  const ref = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(0);

  useEffect(() => {
    const element = ref.current;
    if (!element) {
      return;
    }
    const observer = new ResizeObserver(([entry]) =>
      setWidth(entry.contentRect.width)
    );
    observer.observe(element);
    return () => observer.disconnect();
  }, []);

  return { ref, width };
}

export function MultiAxisLineChart({
  times,
  lines,
  left,
  right,
  label,
  height = 220,
  formatTime = formatElapsed,
  timeTicks,
}: MultiAxisLineChartProps) {
  const { ref, width } = useWidth();
  const [hovered, setHovered] = useState<number>();

  const leftWidth =
    8 + CHAR_WIDTH * Math.max(...left.ticks.map((t) => left.format(t).length));
  const rightWidth = right
    ? 8 +
      CHAR_WIDTH * Math.max(...right.ticks.map((t) => right.format(t).length))
    : 0;
  const plotLeft = MARGIN_SIDE + leftWidth;
  const plotRight = width - MARGIN_SIDE - rightWidth;
  const plotWidth = Math.max(plotRight - plotLeft, 0);
  const plotHeight = height - MARGIN_TOP - MARGIN_BOTTOM;
  const maxT = times.length > 0 ? times[times.length - 1] : 0;

  const x = (t: number) =>
    plotLeft + (maxT > 0 ? (t / maxT) * plotWidth : plotWidth / 2);
  const y = (value: number, axis: LineChartAxis) =>
    MARGIN_TOP +
    plotHeight -
    ((value - axis.min) / (axis.max - axis.min || 1)) * plotHeight;
  const axisOf = (line: LineChartLine) =>
    line.axis === "right" && right ? right : left;

  const paths = lines.map((line) => {
    const axis = axisOf(line);
    const points = times.flatMap((t, i) => {
      const value = line.values[i];
      return value === undefined ? [] : [[x(t), y(value, axis)] as const];
    });
    const d = points
      .map(([px, py], i) => `${i ? "L" : "M"}${px.toFixed(1)} ${py.toFixed(1)}`)
      .join(" ");
    const baseline = y(Math.max(axis.min, Math.min(0, axis.max)), axis);
    const area =
      line.hasArea && points.length > 1
        ? `${d} L${points[points.length - 1][0].toFixed(1)} ${baseline} L${points[0][0].toFixed(1)} ${baseline}Z`
        : undefined;
    return { d: points.length > 1 ? d : undefined, area };
  });

  const xTicks =
    timeTicks ??
    Array.from({ length: X_TICKS }, (_, i) => (i / (X_TICKS - 1)) * maxT);

  const summary = lines
    .map((line) => {
      const values = line.values.filter((v) => v !== undefined);
      return values.length === 0
        ? undefined
        : `${line.name} from ${line.format(Math.min(...values))} to ${line.format(Math.max(...values))}, last ${line.format(values[values.length - 1])}`;
    })
    .filter(Boolean)
    .join("; ");

  const onKeyDown = (event: React.KeyboardEvent<SVGSVGElement>) => {
    const step = { ArrowLeft: -1, ArrowRight: 1 }[event.key];
    if (step === undefined || times.length === 0) {
      return;
    }
    event.preventDefault();
    setHovered((current) =>
      Math.min(times.length - 1, Math.max(0, (current ?? 0) + step))
    );
  };

  const onPointerMove = (event: React.PointerEvent<SVGSVGElement>) => {
    const box = event.currentTarget.getBoundingClientRect();
    const t = ((event.clientX - box.left - plotLeft) / plotWidth) * maxT;
    let nearest = 0;
    times.forEach((time, i) => {
      if (Math.abs(time - t) < Math.abs(times[nearest] - t)) nearest = i;
    });
    setHovered(nearest);
  };

  const hoveredX = hovered === undefined ? 0 : x(times[hovered]);

  return (
    <div ref={ref} className="relative w-full" style={{ height }}>
      {width > 0 && (
        <svg
          width={width}
          height={height}
          role="img"
          aria-label={summary ? `${label}. ${summary}` : label}
          tabIndex={0}
          onKeyDown={onKeyDown}
          onBlur={() => setHovered(undefined)}
          className="block"
          onPointerMove={onPointerMove}
          onPointerLeave={() => setHovered(undefined)}
        >
          {left.ticks.map((tick) => (
            <g key={tick}>
              <line
                x1={plotLeft}
                x2={plotRight}
                y1={y(tick, left)}
                y2={y(tick, left)}
                stroke="var(--border-color-secondary)"
              />
              <text
                x={plotLeft - 8}
                y={y(tick, left) + 4}
                textAnchor="end"
                fontSize={12}
                fill="var(--text-color-quaternary)"
              >
                {left.format(tick)}
              </text>
            </g>
          ))}
          {right?.ticks.map((tick) => (
            <text
              key={tick}
              x={plotRight + 8}
              y={y(tick, right) + 4}
              fontSize={12}
              fill="var(--text-color-quaternary)"
            >
              {right.format(tick)}
            </text>
          ))}
          {xTicks.map((t) => (
            <text
              key={t}
              x={x(t)}
              y={height - 6}
              textAnchor={t === 0 ? "start" : t === maxT ? "end" : "middle"}
              fontSize={12}
              fill="var(--text-color-quaternary)"
            >
              {formatTime(t)}
            </text>
          ))}
          {lines.map((line, i) => (
            <g key={line.name}>
              {paths[i].area && (
                <path d={paths[i].area} fill={line.color} opacity={0.14} />
              )}
              {paths[i].d && (
                <path
                  d={paths[i].d}
                  fill="none"
                  stroke={line.color}
                  strokeWidth={1.8}
                  strokeLinejoin="round"
                  strokeDasharray={line.isDashed ? "4 3" : undefined}
                />
              )}
            </g>
          ))}
          {hovered !== undefined && (
            <>
              <line
                x1={hoveredX}
                x2={hoveredX}
                y1={MARGIN_TOP}
                y2={MARGIN_TOP + plotHeight}
                stroke="var(--text-color-quaternary)"
                strokeDasharray="3 3"
              />
              {lines.map((line) => {
                const value = line.values[hovered];
                return value === undefined ? null : (
                  <circle
                    key={line.name}
                    cx={hoveredX}
                    cy={y(value, axisOf(line))}
                    r={3.5}
                    fill={line.color}
                  />
                );
              })}
            </>
          )}
        </svg>
      )}
      {hovered !== undefined && (
        <div
          className="bg-primary border-secondary text-primary pointer-events-none absolute top-2 z-10 rounded-lg border px-2 py-1 text-xs shadow-lg"
          style={
            hoveredX > width / 2
              ? { right: width - hoveredX + 12 }
              : { left: hoveredX + 12 }
          }
        >
          <div className="text-tertiary">{formatTime(times[hovered])}</div>
          {lines.map((line) => {
            const value = line.values[hovered];
            return value === undefined ? null : (
              <div key={line.name} className="flex items-center gap-1.5">
                <span
                  className="size-2 rounded-full"
                  style={{ backgroundColor: line.color }}
                />
                <span className="tabular-nums">
                  {line.name} {line.format(value)}
                </span>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
