import clsx from "clsx";
import { useState } from "react";
import {
  Input as AriaInput,
  Text,
  TextField,
  TextFieldProps,
} from "react-aria-components";

import { Typography } from "..";
import { FieldLabel } from "../field/field-label";
import { FieldSupportText } from "../field/field-support-text";

export type OTPInputProps = Omit<
  TextFieldProps,
  "type" | "children" | "value" | "onChange"
> & {
  label?: string;
  supportText?: string;
  length?: number;
  value: string;
  onChange: (value: string) => void;
  onComplete?: (value: string) => void;
};

/**
 * One-time code field rendered as a box per digit.
 *
 * The boxes are decoration over a single real input, so pasting, one-time-code
 * autofill and screen readers behave as they would on any text field.
 */
export function OTPInput({
  label,
  supportText,
  length = 6,
  value,
  onChange,
  onComplete,
  onFocusChange,
  className,
  ...props
}: OTPInputProps) {
  const [isFocused, setIsFocused] = useState(false);

  const activeIndex = Math.min(value.length, length - 1);

  const handleFocusChange = (focused: boolean) => {
    setIsFocused(focused);
    onFocusChange?.(focused);
  };

  const handleChange = (next: string) => {
    const digits = next.replace(/\D/g, "").slice(0, length);
    onChange(digits);
    if (digits.length === length) {
      onComplete?.(digits);
    }
  };

  const caretToEnd = (event: React.SyntheticEvent<HTMLInputElement>) => {
    const input = event.currentTarget;
    if (input.selectionStart !== input.value.length) {
      input.setSelectionRange(input.value.length, input.value.length);
    }
  };

  return (
    <TextField
      {...props}
      value={value}
      onChange={handleChange}
      onFocusChange={handleFocusChange}
      className={clsx("group flex flex-col gap-1", className)}
    >
      {label && <FieldLabel isRequired={props.isRequired}>{label}</FieldLabel>}

      <div className="relative flex w-fit gap-2">
        <AriaInput
          inputMode="numeric"
          autoComplete="one-time-code"
          pattern="[0-9]*"
          maxLength={length}
          onSelect={caretToEnd}
          className="absolute inset-0 z-10 cursor-text text-transparent caret-transparent opacity-0 outline-none group-data-disabled:cursor-not-allowed"
        />

        {Array.from({ length }, (_, index) => {
          const isActive = isFocused && index === activeIndex;

          return (
            <Typography
              key={index}
              as="div"
              size="lg"
              weight="medium"
              aria-hidden="true"
              className={clsx(
                "border-primary bg-primary text-primary flex h-12 w-10 items-center justify-center rounded-lg border -outline-offset-1 select-none",
                "group-data-invalid:border-error",
                "group-data-disabled:border-tertiary group-data-disabled:bg-disable group-data-disabled:text-disable",
                isActive && "outline-focus border-accent outline-2"
              )}
            >
              {value[index] ??
                (isActive && (
                  <span className="h-6 w-0.25 animate-[caret-blink_1s_step-end_infinite] rounded-full bg-current" />
                ))}
            </Typography>
          );
        })}
      </div>

      {supportText && (
        <Text slot="description" elementType="div">
          <FieldSupportText>{supportText}</FieldSupportText>
        </Text>
      )}
    </TextField>
  );
}
