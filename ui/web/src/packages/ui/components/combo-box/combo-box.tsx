import clsx from "clsx";
import {
  ComboBox as AriaComboBox,
  Button,
  Input,
  ListBox,
  Popover,
} from "react-aria-components";

import { Icon, IconType, Typography } from "..";
import { FieldLabel } from "../field/field-label";
import { FieldSupportText } from "../field/field-support-text";
import { SelectListBoxItem } from "../select/select-item";
import { SelectLoading } from "../select/select-loading";

export type ComboBoxOption = { id: string; name: string; icon?: IconType };

export type ComboBoxProps = React.ComponentProps<typeof AriaComboBox> & {
  label?: string;
  placeholder?: string;
  supportText?: string;
};

function ComboBoxRoot({
  label,
  placeholder,
  supportText,
  className,
  children,
  ...props
}: ComboBoxProps) {
  return (
    <AriaComboBox
      allowsEmptyCollection
      {...props}
      className={clsx("group flex flex-col gap-1", className)}
    >
      <FieldLabel isRequired={props.isRequired}>{label}</FieldLabel>

      <Typography
        as="div"
        variant="body"
        size="sm"
        className={clsx(
          "text-tertiary flex items-center gap-2",
          !props.isDisabled && !props.isReadOnly && "hover:text-secondary",
          "group-data-disabled:text-disable",
          "group-data-focused:text-primary"
        )}
      >
        <Input
          placeholder={!props.isReadOnly ? placeholder : undefined}
          className={clsx(
            "text-primary w-full min-w-0 rounded-lg py-2 pr-9 pl-3 -outline-offset-1",
            "border-primary bg-primary placeholder:text-tertiary border",
            !props.isDisabled && !props.isReadOnly && "hover:bg-primary-hover",
            "group-data-invalid:border-error",
            "group-data-disabled:border-tertiary group-data-disabled:bg-disable data-disabled:cursor-not-allowed",
            props.isReadOnly
              ? "bg-tertiary border-tertiary outline-none"
              : "group-data-focused:outline-focus group-data-focused:outline-2"
          )}
        />
        <Button
          aria-hidden="true"
          className={clsx(
            "relative -ml-9 flex shrink-0 items-center",
            props.isReadOnly && "invisible"
          )}
        >
          <Icon type="caret-down" size="md" />
          {/* Invisible clickable area to increase hit target */}
          <span
            className="absolute inset-0 -m-2 rounded-full"
            aria-hidden="true"
          />
        </Button>
      </Typography>

      {supportText && <FieldSupportText>{supportText}</FieldSupportText>}

      <Popover
        offset={4}
        containerPadding={8}
        className="border-primary bg-primary min-w-[var(--trigger-width)] rounded-lg border shadow-lg"
      >
        <ListBox
          renderEmptyState={() => (
            <Typography
              as="div"
              variant="body"
              size="sm"
              className="text-tertiary p-2 italic"
            >
              No results found
            </Typography>
          )}
          className="max-h-60 cursor-default overflow-auto p-1 outline-0 [&>*]:mb-1 [&>*:has(+[inert])]:mb-0 [&>[inert]]:mb-0"
        >
          {children}
        </ListBox>
      </Popover>
    </AriaComboBox>
  );
}

export const ComboBox = Object.assign(ComboBoxRoot, {
  Item: SelectListBoxItem,
  Loading: SelectLoading,
});
