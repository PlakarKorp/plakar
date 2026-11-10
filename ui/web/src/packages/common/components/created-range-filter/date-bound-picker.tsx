import type { CalendarDateTime } from "@internationalized/date";
import {
  getLocalTimeZone,
  now,
  parseAbsoluteToLocal,
  toCalendarDateTime,
} from "@internationalized/date";
import { DatePicker } from "@plakar-ui/ui/components";
import { useMemo } from "react";

type DateBoundPickerProps = {
  label: string;
  /** Names the field for assistive technology only, when a group label sits above it. */
  isLabelHidden?: boolean;
  value: string | undefined;
  onChange: (value: string | undefined) => void;
};

export function DateBoundPicker({
  label,
  isLabelHidden = false,
  value,
  onChange,
}: DateBoundPickerProps) {
  const timeZone = getLocalTimeZone();
  const latestBound = useMemo(
    () => toCalendarDateTime(now(timeZone)),
    [timeZone]
  );
  const date = useMemo(
    () => (value ? toCalendarDateTime(parseAbsoluteToLocal(value)) : null),
    [value]
  );

  return (
    <DatePicker<CalendarDateTime>
      label={isLabelHidden ? undefined : label}
      aria-label={isLabelHidden ? label : undefined}
      className="max-md:w-full md:w-64"
      granularity="minute"
      maxValue={latestBound}
      value={date}
      onChange={(next) => onChange(next?.toDate(timeZone).toISOString())}
      isClearable
    />
  );
}
