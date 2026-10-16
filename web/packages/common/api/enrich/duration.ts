import { formatDuration, intervalToDuration } from "date-fns";

// value is in nanoseconds
export function Duration(value: number) {
  const duration = intervalToDuration({
    start: 0,
    // end is in milliseconds
    end: value / 1000000,
  });

  return {
    Value: value,
    toHumanString: () => {
      const ret = formatDuration(duration);
      if (ret === "") {
        return "less than a second";
      }
      return ret;
    },
    toShortString: () => {
      const parts: string[] = [];
      if (duration.years) parts.push(`${duration.years}y`);
      if (duration.months) parts.push(`${duration.months}mo`);
      if (duration.weeks) parts.push(`${duration.weeks}w`);
      if (duration.days) parts.push(`${duration.days}d`);
      if (duration.hours) parts.push(`${duration.hours}h`);
      if (duration.minutes) parts.push(`${duration.minutes}m`);
      if (duration.seconds) parts.push(`${duration.seconds}s`);
      return parts.length > 0 ? parts.join("") : "<1s";
    },
    // Called by JSON.stringify to convert the object back to a string
    toJSON() {
      return value;
    },
  };
}
