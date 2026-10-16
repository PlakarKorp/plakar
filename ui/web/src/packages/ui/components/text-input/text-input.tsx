import clsx from "clsx";
import { useState } from "react";
import {
  TextField,
  Input as AriaInput,
  TextFieldProps,
} from "react-aria-components";

import { Icon, IconType, Typography } from "..";
import { FieldButton } from "../field/field-button";
import { FieldLabel } from "../field/field-label";
import { FieldSupportText } from "../field/field-support-text";

type CommonProps = {
  label?: string;
  placeholder?: string;
  icon?: IconType;
  supportText?: string;
  suffix?: React.ReactNode;
  beforeLabelAction?: React.ReactNode;
  labelAction?: React.ReactNode;
};

export type TextInputProps = CommonProps & TextFieldProps;

export function TextInput({
  label,
  placeholder,
  icon,
  supportText,
  suffix,
  beforeLabelAction,
  labelAction,
  className,
  type,
  ...props
}: TextInputProps) {
  const isRevealable = type === "password";
  const [isHidden, setIsHidden] = useState(isRevealable);
  // Nothing to reveal in a read-only field with no value, so hide the toggle.
  const canReveal =
    isRevealable &&
    (!props.isReadOnly || !!(props.value ?? props.defaultValue));

  return (
    <TextField
      {...props}
      type={isRevealable ? (isHidden ? "password" : "text") : type}
      className={clsx("group", className)}
    >
      <div className="flex flex-col gap-1">
        {label && (
          <FieldLabel
            isRequired={props.isRequired}
            beforeAction={beforeLabelAction}
            action={labelAction}
          >
            {label}
          </FieldLabel>
        )}

        <Input
          placeholder={placeholder}
          icon={icon}
          suffix={
            canReveal ? (
              <>
                <FieldButton
                  icon={isHidden ? "eye-slash" : "eye"}
                  aria-label={isHidden ? "Show" : "Hide"}
                  onPress={() => setIsHidden((prev) => !prev)}
                  className="-mr-2"
                />
                {suffix}
              </>
            ) : (
              suffix
            )
          }
          isDisabled={props.isDisabled}
          isReadOnly={props.isReadOnly}
        />

        {supportText && <FieldSupportText>{supportText}</FieldSupportText>}
      </div>
    </TextField>
  );
}

type InputProps = CommonProps & {
  isDisabled?: boolean;
  isReadOnly?: boolean;
  className?: string;
};

export function Input({
  isDisabled,
  isReadOnly,
  placeholder,
  icon,
  suffix,
  className,
}: InputProps) {
  return (
    <Typography
      as="div"
      size="sm"
      className={clsx(
        "border-primary bg-primary text-primary flex items-center gap-2 rounded-lg border px-3 -outline-offset-1",
        !isDisabled &&
          !isReadOnly &&
          "hover:bg-primary-hover hover:text-secondary",
        "group-data-invalid:border-error",
        "group-data-disabled:border-tertiary group-data-disabled:bg-disable group-data-disabled:text-disable",
        isReadOnly
          ? "bg-tertiary text-primary border-tertiary"
          : "has-data-focused:outline-focus has-data-focused:outline-2",
        className
      )}
    >
      {icon && <Icon type={icon} size="md" aria-hidden="true" />}
      <AriaInput
        placeholder={!isReadOnly ? placeholder : undefined}
        className="placeholder:text-tertiary w-full py-2 outline-none [&::-webkit-search-cancel-button]:appearance-none [&::-webkit-search-decoration]:appearance-none"
      />
      {suffix && <span className="inline-flex">{suffix}</span>}
    </Typography>
  );
}
