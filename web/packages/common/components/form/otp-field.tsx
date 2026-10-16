import { OTPInput, OTPInputProps } from "@plakar-ui/ui/components";

import { useFieldContext } from "../../hooks/form-context";
import { useFormReadOnly } from "../../hooks/use-form-read-only";

type OTPFieldProps = Omit<
  OTPInputProps,
  "value" | "onChange" | "onBlur" | "isInvalid"
>;

export function OTPField({ supportText, ...props }: OTPFieldProps) {
  const field = useFieldContext<string>();
  const isReadOnly = useFormReadOnly();

  return (
    <OTPInput
      value={field.state.value ?? ""}
      onChange={field.handleChange}
      onBlur={field.handleBlur}
      isInvalid={field.state.meta.errors.length > 0}
      supportText={field.state.meta.errors[0] || supportText}
      {...props}
      isReadOnly={isReadOnly || props.isReadOnly}
    />
  );
}
