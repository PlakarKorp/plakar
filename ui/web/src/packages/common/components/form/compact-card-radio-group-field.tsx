import {
  CompactCardRadio,
  RadioGroup,
  RadioGroupProps,
  Typography,
} from "@plakar-ui/ui/components";
import { ReactNode } from "react";

import { useFieldContext } from "../../hooks/form-context";
import { useFormReadOnly } from "../../hooks/use-form-read-only";

export function CompactCardRadioGroupField({
  supportText,
  items,
  emptyMessage,
  ...props
}: Omit<RadioGroupProps, "value" | "onChange" | "onBlur" | "isInvalid"> & {
  items: {
    value: string;
    label: string;
    description?: string;
    isDisabled?: boolean;
  }[];
  emptyMessage?: ReactNode;
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
      {items.map(({ value, label, description, isDisabled }) => (
        <CompactCardRadio
          key={value}
          value={value}
          label={label}
          description={description}
          isDisabled={isDisabled}
        />
      ))}
      {items.length === 0 && (
        <Typography size="sm" className="text-tertiary italic">
          {emptyMessage}
        </Typography>
      )}
    </RadioGroup>
  );
}
