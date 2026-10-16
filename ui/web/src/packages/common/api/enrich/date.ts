import {
  formatDistanceToNow,
  format,
  isToday,
  isYesterday,
  differenceInCalendarDays,
  isThisYear,
} from "date-fns";

export type EnrichedDate = ReturnType<typeof Date>;

export function Date(value: string) {
  const date = new globalThis.Date(value);

  return {
    Value: value,
    Date: date,

    relativeToNow: () => {
      return formatDistanceToNow(date, { addSuffix: true });
    },

    toHumanString: () => {
      if (date.getTime() === 0) {
        return undefined;
      }
      const year = date.getFullYear();
      const month = String(date.getMonth() + 1).padStart(2, "0"); // Months are zero-indexed, so add 1
      const day = String(date.getDate()).padStart(2, "0");
      const hours = String(date.getHours()).padStart(2, "0");
      const minutes = String(date.getMinutes()).padStart(2, "0");
      const seconds = String(date.getSeconds()).padStart(2, "0");
      return `${year}/${month}/${day} ${hours}:${minutes}:${seconds}`;
    },

    toShortString: () => {
      if (date.getTime() === 0) {
        return undefined;
      }
      const month = String(date.getMonth() + 1).padStart(2, "0"); // Months are zero-indexed, so add 1
      const day = String(date.getDate()).padStart(2, "0");
      return `${month}/${day}`;
    },

    toFriendlyString: () => {
      if (date.getTime() === 0) return undefined;

      if (isToday(date)) {
        return `Today at ${format(date, "h:mm a")}`;
      }

      if (isYesterday(date)) {
        return `Yesterday at ${format(date, "h:mm a")}`;
      }

      const daysAgo = differenceInCalendarDays(new globalThis.Date(), date);

      // Recent -> "2 days ago"
      if (daysAgo <= 7) {
        return formatDistanceToNow(date, { addSuffix: true });
      }

      // Older this year -> "Mar 12"
      if (isThisYear(date)) {
        return format(date, "MMM d");
      }

      // Older -> "Mar 12, 2023"
      return format(date, "MMM d, yyyy");
    },

    format: (fmt: string) => format(date, fmt),

    // Called by JSON.stringify to convert the object back to a string
    toJSON() {
      return value;
    },
  };
}
