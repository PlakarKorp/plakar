import clsx from "clsx";
import { useContext } from "react";
import {
  Calendar,
  CalendarCell,
  CalendarGrid,
  DateInput,
  DatePicker as AriaDatePicker,
  DatePickerStateContext,
  DateSegment,
  Dialog,
  Group,
  Heading,
  Popover,
  DatePickerProps as AriaDatePickerProps,
  DateValue,
  CalendarGridHeader,
  CalendarGridBody,
  CalendarHeaderCell,
} from "react-aria-components";

import { FieldButton } from "../field/field-button";
import { FieldLabel } from "../field/field-label";
import { FieldSupportText } from "../field/field-support-text";
import { IconButton } from "../icon-button/icon-button";
import { Typography } from "../typography/typography";

export type DatePickerProps<T extends DateValue> = AriaDatePickerProps<T> & {
  label?: string;
  supportText?: string;
  isClearable?: boolean;
};

// Reads the state rather than the value prop, so it clears whether the caller
// holds the value or the picker does. Its slot is nulled or React Aria takes
// the button for the calendar's trigger.
function ClearButton({ isDisabled }: { isDisabled?: boolean }) {
  const state = useContext(DatePickerStateContext);

  if (!state?.value) {
    return null;
  }

  return (
    <FieldButton
      slot={null}
      icon="close"
      aria-label="Clear date"
      isDisabled={isDisabled}
      onPress={() => state.setValue(null)}
    />
  );
}

export function DatePicker<T extends DateValue>({
  label,
  supportText,
  className,
  isClearable,
  ...props
}: DatePickerProps<T>) {
  return (
    <AriaDatePicker
      {...props}
      className={clsx("group flex flex-col gap-1", className)}
    >
      <FieldLabel isRequired={props.isRequired}>{label}</FieldLabel>

      <Group>
        <Typography
          as="div"
          variant="body"
          size="sm"
          className={clsx(
            "text-primary flex items-center justify-between gap-2",
            "border-primary bg-primary rounded-lg border px-2 -outline-offset-1",
            !props.isDisabled &&
              !props.isReadOnly &&
              "hover:bg-primary-hover hover:text-secondary",
            "group-data-invalid:border-error",
            "group-data-disabled:border-tertiary group-data-disabled:bg-disable group-data-disabled:text-disable group-data-disabled:cursor-not-allowed",
            props.isReadOnly
              ? "bg-tertiary border-tertiary"
              : "has-data-focused:outline-focus has-data-focused:outline-2",
            "group-data-open:border-accent-heavy"
          )}
        >
          <DateInput className="py-2">
            {(segment) => (
              <DateSegment
                segment={segment}
                className={clsx(
                  "rounded-sm py-0.5 outline-none",
                  "[&[role=spinbutton]]:px-1",
                  "aria-[valuetext='Empty']:text-placeholder",
                  props.isReadOnly && !props.value && "invisible",
                  !props.isReadOnly &&
                    "data-focused:bg-focus data-focused:text-on-accent"
                )}
              />
            )}
          </DateInput>

          {/* One last child, or justify-between strands the clear button
              halfway across the field. */}
          <span className="flex items-center -space-x-2">
            {isClearable && !props.isReadOnly && (
              <ClearButton isDisabled={props.isDisabled} />
            )}

            <FieldButton
              icon="caret-down"
              aria-label="Toggle calendar"
              className={clsx("-mr-1", props.isReadOnly && "invisible")}
            />
          </span>
        </Typography>
      </Group>

      {supportText && <FieldSupportText>{supportText}</FieldSupportText>}

      <Popover
        offset={4}
        containerPadding={8}
        className="border-primary bg-primary rounded-lg border p-4 shadow-lg"
      >
        <Dialog>
          <Calendar>
            <header className="mb-2 flex items-center justify-between gap-1">
              <IconButton
                slot="previous"
                icon="chevron-left"
                size="md"
                variant="secondary"
              />
              <Heading className="text-primary text-md font-medium" />
              <IconButton
                slot="next"
                icon="chevron-right"
                size="md"
                variant="secondary"
              />
            </header>
            <CalendarGrid
              weekdayStyle="short"
              className="-mx-1 -mb-1 border-separate border-spacing-2"
            >
              <CalendarGridHeader>
                {(day) => (
                  <CalendarHeaderCell
                    children={day}
                    className="text-tertiary text-sm font-medium"
                  />
                )}
              </CalendarGridHeader>
              <CalendarGridBody>
                {(date) => (
                  <CalendarCell
                    date={date}
                    className={clsx(
                      "flex cursor-pointer items-center justify-center",
                      "outline-focus h-8 w-8 rounded-full text-sm outline-offset-2",
                      "aria-disabled:hidden",
                      "hover:bg-primary-hover",
                      "data-selected:bg-accent data-selected:text-on-accent"
                    )}
                  />
                )}
              </CalendarGridBody>
            </CalendarGrid>
          </Calendar>
        </Dialog>
      </Popover>
    </AriaDatePicker>
  );
}
