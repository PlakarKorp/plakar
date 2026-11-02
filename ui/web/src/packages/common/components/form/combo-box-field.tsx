import {
  ComboBox,
  ComboBoxOption,
  ComboBoxProps,
  Select,
} from "@plakar-ui/ui/components";

import { useFieldContext } from "../../hooks/form-context";
import { useFormReadOnly } from "../../hooks/use-form-read-only";

export type ComboBoxSection = { title: string; options: ComboBoxOption[] };

type ComboBoxFieldProps = Omit<
  ComboBoxProps,
  "selectedKey" | "onSelectionChange" | "onBlur" | "isInvalid"
> &
  (
    | { options: ComboBoxOption[]; sections?: never }
    | { sections: ComboBoxSection[]; options?: never }
  );

export function ComboBoxField({
  options,
  sections,
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
      {sections
        ? sections.map((section) => (
            <Select.Section key={section.title} title={section.title}>
              {section.options.map(({ id, name }) => (
                <ComboBox.Item key={id} id={id} name={name} />
              ))}
            </Select.Section>
          ))
        : options.map(({ id, name }) => (
            <ComboBox.Item key={id} id={id} name={name} />
          ))}
    </ComboBox>
  );
}
