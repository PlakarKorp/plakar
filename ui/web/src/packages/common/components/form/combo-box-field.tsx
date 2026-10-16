import {
  ComboBox,
  ComboBoxOption,
  ComboBoxProps,
} from "@plakar-ui/ui/components";

import { useFieldContext } from "../../hooks/form-context";
import { useFormReadOnly } from "../../hooks/use-form-read-only";

type ComboBoxFieldProps = Omit<
  ComboBoxProps,
  "selectedKey" | "onSelectionChange" | "onBlur" | "isInvalid"
> & { options: ComboBoxOption[] };

export function ComboBoxField({
  options,
  supportText,
  ...props
}: ComboBoxFieldProps) {
  const field = useFieldContext<string>();
  const isReadOnly = useFormReadOnly();

  return (
    <ComboBox
      selectedKey={field.state.value || null}
      onSelectionChange={(key) => field.handleChange(key ? String(key) : "")}
      onBlur={field.handleBlur}
      isInvalid={field.state.meta.errors.length > 0}
      supportText={field.state.meta.errors[0] || supportText}
      {...props}
      isReadOnly={isReadOnly || props.isReadOnly}
    >
      {options.map(({ id, name }) => (
        <ComboBox.Item key={id} id={id} name={name} />
      ))}
    </ComboBox>
  );
}
