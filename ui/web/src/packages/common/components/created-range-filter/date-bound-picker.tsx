import type { CalendarDateTime } from "@internationalized/date";
import {
  getLocalTimeZone,
  now,
  parseAbsoluteToLocal,
  toCalendarDateTime,
} from "@internationalized/date";
import { DatePicker } from "@plakar-ui/ui/components";
import { useMemo } from "react";

const END_OF_MINUTE = { second: 59, millisecond: 999 };
const END_OF_DAY = { hour: 23, minute: 59, ...END_OF_MINUTE };

type DateBoundPickerProps = {
  label: string;
  /** Names the field for assistive technology only, when a group label sits above it. */
  isLabelHidden?: boolean;
  /** Ends a range: a day picked from the calendar runs to 23:59, and the bound includes its last minute. */
  isEnd?: boolean;
  value: string | undefined;
  onChange: (value: string | undefined) => void;
};

export function DateBoundPicker({
  label,
  isLabelHidden = false,
  isEnd = false,
  value,
  onChange,
}: DateBoundPickerProps) {
  const timeZone = getLocalTimeZone();
  const latestBound = useMemo(() => {
    const today = toCalendarDateTime(now(timeZone));
    return isEnd ? today.set(END_OF_DAY) : today;
  }, [timeZone, isEnd]);
  const date = useMemo(
    () => (value ? toCalendarDateTime(parseAbsoluteToLocal(value)) : null),
    [value]
  );

  const toValue = (next: CalendarDateTime) =>
    (isEnd ? next.set(END_OF_MINUTE) : next).toDate(timeZone).toISOString();

  return (
    <DatePicker<CalendarDateTime>
      label={isLabelHidden ? undefined : label}
      aria-label={isLabelHidden ? label : undefined}
      className="max-md:w-full md:w-64"
      granularity="minute"
      maxValue={latestBound}
      placeholderValue={isEnd ? latestBound : undefined}
      value={date}
      onChange={(next) => onChange(next ? toValue(next) : undefined)}
      isClearable
    />
  );
}
