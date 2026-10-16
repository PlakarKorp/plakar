import {
  FieldButton,
  TextInput,
  TextInputProps,
} from "@plakar-ui/ui/components";

import { useFieldContext } from "../../hooks/form-context";
import { useFormReadOnly } from "../../hooks/use-form-read-only";

type TextFieldProps = Omit<
  TextInputProps,
  "value" | "onChange" | "onBlur" | "isInvalid"
> & {
  generateValue?: () => string;
};

export function TextField({
  type,
  supportText,
  generateValue,
  ...props
}: TextFieldProps) {
  const field = useFieldContext<string>();

  const isReadOnly = useFormReadOnly() || props.isReadOnly;

  const canGenerate = !!generateValue && !isReadOnly;

  const handleGenerate =
    generateValue && !isReadOnly
      ? () => field.handleChange(generateValue())
      : undefined;

  return (
    <TextInput
      value={field.state.value ?? props.defaultValue}
      type={type}
      onChange={field.handleChange}
      onBlur={field.handleBlur}
      isInvalid={field.state.meta.errors.length > 0}
      supportText={field.state.meta.errors[0] || supportText}
      suffix={
        canGenerate && (
          <FieldButton
            icon="wand-magic-sparkles"
            aria-label={
              props.label ? `Generate ${props.label}` : "Generate value"
            }
            onPress={handleGenerate}
            className="-mr-2"
          />
        )
      }
      {...props}
      isReadOnly={isReadOnly}
    />
  );
}
