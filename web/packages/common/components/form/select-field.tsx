import { Select, SelectProps, SelectOption } from "@plakar-ui/ui/components";

import { useFieldContext } from "../../hooks/form-context";
import { useFormReadOnly } from "../../hooks/use-form-read-only";

export function SelectField({
  options,
  isSearchable = false,
  supportText,
  ...props
}: Omit<SelectProps, "value" | "onChange" | "onBlur" | "isInvalid"> & {
  options: readonly SelectOption[];
  isSearchable?: boolean;
}) {
  const field = useFieldContext<string>();
  const isReadOnly = useFormReadOnly();

  const ListBox = isSearchable ? Select.SearchableListBox : Select.ListBox;

  return (
    <Select
      value={field.state.value ?? null}
      onChange={(key) => {
        field.handleChange(key ? String(key) : "");
      }}
      onBlur={field.handleBlur}
      isInvalid={field.state.meta.errors.length > 0}
      supportText={field.state.meta.errors[0] || supportText}
      {...props}
      isReadOnly={isReadOnly || props.isReadOnly}
    >
      <ListBox>
        {options.map((option) => (
          <Select.Item key={option.id} {...option} />
        ))}
      </ListBox>
    </Select>
  );
}
