import clsx from "clsx";
import {
  DateInput,
  DateSegment,
  TimeField as AriaTimeField,
  TimeFieldProps as AriaTimeFieldProps,
  TimeValue,
} from "react-aria-components";

import { FieldLabel } from "../field/field-label";
import { FieldSupportText } from "../field/field-support-text";
import { Typography } from "../typography/typography";

export type TimeInputProps<T extends TimeValue> = AriaTimeFieldProps<T> & {
  label?: string;
  supportText?: string;
  inputWidth?: string;
};

export function TimeInput<T extends TimeValue>({
  label,
  supportText,
  className,
  inputWidth = "w-full",
  ...props
}: TimeInputProps<T>) {
  return (
    <AriaTimeField
      hourCycle={24}
      shouldForceLeadingZeros
      {...props}
      className={clsx("group flex flex-col gap-1", className)}
    >
      {label && <FieldLabel isRequired={props.isRequired}>{label}</FieldLabel>}

      <Typography
        as="div"
        variant="body"
        size="sm"
        className={clsx(
          "text-primary flex h-9.5 items-center rounded-lg border px-2 -outline-offset-1",
          inputWidth,
          "border-primary bg-primary",
          !props.isDisabled && !props.isReadOnly && "hover:bg-primary-hover",
          "group-data-invalid:border-error",
          "group-data-disabled:border-tertiary group-data-disabled:bg-disable group-data-disabled:text-disable group-data-disabled:cursor-not-allowed",
          props.isReadOnly
            ? "bg-tertiary border-tertiary"
            : "has-data-focused:outline-focus has-data-focused:outline-2"
        )}
      >
        <DateInput className="flex">
          {(segment) => (
            <DateSegment
              segment={segment}
              className={clsx(
                "rounded-sm py-0.5 tabular-nums outline-none",
                "[&[role=spinbutton]]:px-1",
                "aria-[valuetext='Empty']:text-placeholder",
                props.isReadOnly && !props.value && "invisible",
                !props.isReadOnly &&
                  "data-focused:bg-focus data-focused:text-on-accent"
              )}
            />
          )}
        </DateInput>
      </Typography>

      {supportText && <FieldSupportText>{supportText}</FieldSupportText>}
    </AriaTimeField>
  );
}
