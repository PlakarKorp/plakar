import clsx from "clsx";
import {
  TextField,
  TextFieldProps,
  TextArea as AriaTextArea,
} from "react-aria-components";

import { FieldLabel } from "../field/field-label";
import { FieldSupportText } from "../field/field-support-text";

type TextAreaProps = TextFieldProps & {
  label?: string;
  placeholder?: string;
  supportText?: string;
  suffix?: React.ReactNode;
  rows?: number;
  beforeLabelAction?: React.ReactNode;
  labelAction?: React.ReactNode;
};

export function TextArea({
  className,
  label,
  supportText,
  placeholder,
  suffix,
  rows,
  beforeLabelAction,
  labelAction,
  ...props
}: TextAreaProps) {
  return (
    <div className={clsx("relative", className)}>
      <TextField {...props} className="group">
        <div className="flex flex-col gap-1">
          {(label || beforeLabelAction || labelAction) && (
            <FieldLabel
              isRequired={props.isRequired}
              beforeAction={beforeLabelAction}
              action={labelAction}
            >
              {label}
            </FieldLabel>
          )}

          <AriaTextArea
            placeholder={!props.isReadOnly ? placeholder : undefined}
            rows={rows}
            className={clsx(
              "border-primary bg-primary rounded-lg border px-3 py-2 outline-offset-1",
              "text-primary placeholder:text-tertiary text-sm",
              !props.isDisabled &&
                !props.isReadOnly &&
                "hover:bg-primary-hover",
              "group-data-invalid:border-error",
              "group-data-disabled:border-tertiary group-data-disabled:bg-disable group-data-disabled:text-disable",
              props.isReadOnly
                ? "bg-tertiary text-primary border-tertiary"
                : "has-data-focused:outline-focus has-data-focused:outline-2"
            )}
          />

          {supportText && <FieldSupportText>{supportText}</FieldSupportText>}
        </div>
      </TextField>

      <div className="absolute top-0 right-0">{suffix}</div>
    </div>
  );
}
