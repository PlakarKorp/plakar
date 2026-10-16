import { Radio, RadioGroup, RadioGroupProps } from "@plakar-ui/ui/components";

import { useFieldContext } from "../../hooks/form-context";
import { useFormReadOnly } from "../../hooks/use-form-read-only";

export function RadioGroupField({
  supportText,
  items,
  ...props
}: Omit<RadioGroupProps, "value" | "onChange" | "onBlur" | "isInvalid"> & {
  items: { value: string; label: string }[];
}) {
  const field = useFieldContext<string>();
  const isReadOnly = useFormReadOnly();

  return (
    <RadioGroup
      value={field.state.value ?? ""}
      onChange={field.handleChange}
      onBlur={field.handleBlur}
      isInvalid={field.state.meta.errors.length > 0}
      supportText={field.state.meta.errors[0] || supportText}
      {...props}
      isDisabled={isReadOnly || props.isDisabled}
    >
      {items.map(({ value, label }) => (
        <Radio key={value} value={value}>
          {label}
        </Radio>
      ))}
    </RadioGroup>
  );
}
