import { Time } from "@internationalized/date";
import { TimeInput, TimeInputProps } from "@plakar-ui/ui/components";

import { useFieldContext } from "../../hooks/form-context";
import { useFormReadOnly } from "../../hooks/use-form-read-only";

export function TimeField({
  supportText,
  ...props
}: Omit<TimeInputProps<Time>, "value" | "onChange" | "onBlur" | "isInvalid">) {
  const field = useFieldContext<Time | null>();
  const isReadOnly = useFormReadOnly();

  return (
    <TimeInput
      value={field.state.value}
      onChange={field.handleChange}
      onBlur={field.handleBlur}
      isInvalid={field.state.meta.errors.length > 0}
      supportText={field.state.meta.errors[0] || supportText}
      {...props}
      isReadOnly={isReadOnly || props.isReadOnly}
    />
  );
}
