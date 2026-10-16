import { NumberInput, NumberInputProps } from "@plakar-ui/ui/components";

import { useFieldContext } from "../../hooks/form-context";
import { useFormReadOnly } from "../../hooks/use-form-read-only";

export function NumberField({
  supportText,
  ...props
}: Omit<NumberInputProps, "value" | "onChange" | "onBlur" | "isInvalid">) {
  const field = useFieldContext<number>();
  const isReadOnly = useFormReadOnly();

  return (
    <NumberInput
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
