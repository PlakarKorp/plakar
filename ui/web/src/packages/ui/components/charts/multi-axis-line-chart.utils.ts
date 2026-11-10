export type LineChartAxis = {
  min: number;
  max: number;
  ticks: number[];
  format: (value: number) => string;
};

export type LineChartLine = {
  name: string;
  color: string;
  values: (number | undefined)[];
  axis: "left" | "right";
  format: (value: number) => string;
  hasArea?: boolean;
  isDashed?: boolean;
};

export const SERIES_ACCENT = "var(--background-color-accent)";
export const SERIES_NEUTRAL = "var(--text-color-tertiary)";
export const SERIES_REFERENCE = "var(--text-color-secondary)";

/**
 * Returns a round step, in 1, 2, 2.5, 5 or 10 times a power of ten, that is not smaller than the raw step.
 */
export function niceStep(raw: number): number {
  if (raw <= 0) {
    return 1;
  }
  const power = 10 ** Math.floor(Math.log10(raw));
  const fraction = raw / power;
  const factor = [1, 2, 2.5, 5, 10].find((f) => fraction <= f) ?? 10;
  return factor * power;
}

export function evenTicks(min: number, step: number, count: number) {
  return Array.from({ length: count + 1 }, (_, i) => min + i * step);
}
