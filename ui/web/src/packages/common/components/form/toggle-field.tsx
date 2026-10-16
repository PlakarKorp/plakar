import { Toggle, ToggleProps } from "@plakar-ui/ui/components";

import { useFieldContext } from "../../hooks/form-context";
import { useFormReadOnly } from "../../hooks/use-form-read-only";

export function ToggleField(props: Omit<ToggleProps, "onChange" | "onBlur">) {
  const field = useFieldContext<boolean | "true" | "false">();
  const isReadOnly = useFormReadOnly();

  const value = field.state.value;
  const isSelected = value === true || value === "true";

  const handleChange = (newValue: boolean) => {
    // Keep field value type consistent: if it was a string before, store "true"/"false". If it was a boolean, store a boolean.
    field.handleChange(
      typeof value === "string" ? (newValue ? "true" : "false") : newValue
    );
  };

  return (
    <Toggle
      title={props.title}
      isSelected={isSelected}
      onChange={handleChange}
      onBlur={field.handleBlur}
      {...props}
      isDisabled={isReadOnly || props.isDisabled}
    />
  );
}
