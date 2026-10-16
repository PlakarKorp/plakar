import { CalendarDateTime } from "@internationalized/date";
import { DatePicker, DatePickerProps } from "@plakar-ui/ui/components";

import { useFieldContext } from "../../hooks/form-context";
import { useFormReadOnly } from "../../hooks/use-form-read-only";

export function DateField({
  supportText,
  ...props
}: Omit<
  DatePickerProps<CalendarDateTime>,
  "value" | "onChange" | "onBlur" | "isInvalid"
>) {
  const field = useFieldContext<CalendarDateTime | null>();
  const isReadOnly = useFormReadOnly();

  return (
    <DatePicker
      value={field.state.value ?? null}
      onChange={field.handleChange}
      onBlur={field.handleBlur}
      isInvalid={field.state.meta.errors.length > 0}
      supportText={field.state.meta.errors[0] || supportText}
      {...props}
      isReadOnly={isReadOnly || props.isReadOnly}
    />
  );
}
