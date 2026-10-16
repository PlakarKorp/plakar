import clsx from "clsx";
import {
  NumberField,
  Input,
  NumberFieldProps,
  Group,
  Button,
} from "react-aria-components";

import { Icon } from "..";
import { FieldLabel } from "../field/field-label";
import { FieldSupportText } from "../field/field-support-text";

export type NumberInputProps = NumberFieldProps & {
  label?: string;
  placeholder?: string;
  supportText?: string;
  inputWidth?: string;
  labelAction?: React.ReactNode;
};

export function NumberInput({
  label,
  placeholder,
  supportText,
  className,
  inputWidth = "w-full",
  labelAction,
  ...props
}: NumberInputProps) {
  return (
    <NumberField {...props} className={clsx("group", className)}>
      <div className="flex flex-col gap-1">
        {label && (
          <FieldLabel isRequired={props.isRequired} action={labelAction}>
            {label}
          </FieldLabel>
        )}

        <Group
          className={clsx(
            "group/input text-primary flex h-9.5 rounded-lg",
            props.isReadOnly
              ? "bg-tertiary"
              : "bg-primary hover:bg-primary-hover",
            inputWidth
          )}
        >
          {!props.isReadOnly && <ActionButton slot="decrement" />}

          <Input
            placeholder={props.isReadOnly ? undefined : placeholder}
            className={clsx(
              "z-0 h-full w-full border px-3 outline-0 -outline-offset-1",
              props.isReadOnly
                ? "border-tertiary rounded-lg"
                : "border-primary group-focus-within/input:outline-focus group-focus-within/input:outline-2",
              "group-data-invalid:border-error group-data-invalid:text-primary",
              "group-data-disabled:border-background-disable group-data-disabled:bg-disable group-data-disabled:text-disable"
            )}
          />

          {!props.isReadOnly && <ActionButton slot="increment" />}
        </Group>

        {supportText && <FieldSupportText>{supportText}</FieldSupportText>}
      </div>
    </NumberField>
  );
}

type ActionButtonProps = {
  slot: "decrement" | "increment";
};

function ActionButton({ slot }: ActionButtonProps) {
  return (
    <Button
      slot={slot}
      className={clsx(
        "border-primary flex h-full items-center justify-center border px-3 -outline-offset-1",
        {
          increment: "outline-l-0 rounded-r-lg border-l-0",
          decrement: "outline-r-0 rounded-l-lg border-r-0",
        }[slot],
        "group-focus-within/input:outline-focus group-focus-within/input:outline-2",
        "group-data-invalid:border-error group-data-invalid:text-primary",
        "group-data-disabled:border-background-disable group-data-disabled:bg-disable group-data-disabled:text-disable data-disabled:border-background-disable data-disabled:bg-disable data-disabled:text-disable data-disabled:outline-0"
      )}
    >
      <Icon type={slot === "increment" ? "plus" : "minus"} size="sm" />
    </Button>
  );
}
