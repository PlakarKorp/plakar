import clsx from "clsx";

import { Typography } from "../typography/typography";

export type ActivityDay = {
  date: Date;
  count: number;
};

type ActivityHeatmapProps = {
  days: ActivityDay[];
};

const WEEKDAY_LABELS = ["M", "T", "W", "T", "F", "S", "S"];

const LEVEL_CLASSES = [
  "bg-tertiary",
  "bg-accent-tertiary",
  "bg-accent-secondary",
  "bg-accent",
];

function levelClass(count: number) {
  return LEVEL_CLASSES[Math.min(count, LEVEL_CLASSES.length - 1)];
}

function mondayBasedWeekday(date: Date) {
  return (date.getDay() + 6) % 7;
}

function dayLabel({ date, count }: ActivityDay) {
  return `${date.toDateString()}: ${count}`;
}

function levelLabel(level: number) {
  return level === LEVEL_CLASSES.length - 1 ? `${level}+` : `${level}`;
}

function shortDate(date: Date) {
  if (date.toDateString() === new Date().toDateString()) {
    return "Today";
  }
  return date.toLocaleDateString("en-US", { month: "short", day: "numeric" });
}

function periodLabel(days: ActivityDay[]) {
  if (days.length === 0) {
    return "";
  }
  return `${shortDate(days[0].date)} → ${shortDate(days[days.length - 1].date)}`;
}

export function ActivityHeatmap({ days }: ActivityHeatmapProps) {
  const leadingBlanks = days.length > 0 ? mondayBasedWeekday(days[0].date) : 0;

  return (
    <div className="flex flex-col gap-3">
      <div className="grid grid-cols-7 gap-1">
        {WEEKDAY_LABELS.map((label, index) => (
          <Typography
            key={index}
            as="span"
            size="xs"
            aria-hidden="true"
            className="text-tertiary text-center"
          >
            {label}
          </Typography>
        ))}
      </div>
      <ol className="grid grid-cols-7 gap-1">
        {Array.from({ length: leadingBlanks }, (_, index) => (
          <li key={`blank-${index}`} aria-hidden="true" />
        ))}
        {days.map((day) => (
          <li
            key={day.date.toISOString()}
            title={dayLabel(day)}
            aria-label={dayLabel(day)}
            className={clsx("h-5 rounded", levelClass(day.count))}
          />
        ))}
      </ol>
      <div
        aria-hidden="true"
        className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1"
      >
        <Typography as="span" size="xs" className="text-tertiary">
          {periodLabel(days)}
        </Typography>
        <span className="flex items-center gap-2.5">
          {LEVEL_CLASSES.map((className, level) => (
            <span key={className} className="flex items-center gap-1">
              <span className={clsx("size-2.5 rounded-xs", className)} />
              <Typography as="span" size="xs" className="text-tertiary">
                {levelLabel(level)}
              </Typography>
            </span>
          ))}
        </span>
      </div>
    </div>
  );
}
