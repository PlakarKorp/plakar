import {
  MultiSelect,
  MultiSelectOption,
  MultiSelectProps,
} from "@plakar-ui/ui/components";

import { useFieldContext } from "../../hooks/form-context";
import { useFormReadOnly } from "../../hooks/use-form-read-only";

type MultiSelectFieldProps = Omit<
  MultiSelectProps,
  "value" | "onChange" | "onBlur" | "isInvalid"
> & { options: MultiSelectOption[] };

export function MultiSelectField({
  options,
  supportText,
  ...props
}: MultiSelectFieldProps) {
  const field = useFieldContext<string[]>();
  const isReadOnly = useFormReadOnly();

  return (
    <MultiSelect
      value={field.state.value ?? []}
      onChange={(keys) =>
        field.handleChange(keys.length ? keys.map(String) : [])
      }
      onBlur={field.handleBlur}
      isInvalid={field.state.meta.errors.length > 0}
      supportText={field.state.meta.errors[0] || supportText}
      {...props}
      isDisabled={isReadOnly || props.isDisabled}
    >
      <MultiSelect.ListBox>
        {options.map((option) => (
          <MultiSelect.Item key={option.id} {...option} />
        ))}
      </MultiSelect.ListBox>
    </MultiSelect>
  );
}
