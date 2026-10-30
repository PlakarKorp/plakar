import clsx from "clsx";
import { ReactNode } from "react";
import {
  Button,
  Dialog,
  DialogTrigger,
  Heading,
  Popover as AriaPopover,
} from "react-aria-components";

import { FieldLabel } from "../field/field-label";
import { FieldSupportText } from "../field/field-support-text";
import { Icon } from "../icon/icon";
import { Typography } from "../typography/typography";

export type FieldPopoverProps = {
  value: ReactNode;
  children: ReactNode;
  title?: string;
  footer?: ReactNode;
  label?: string;
  placeholder?: string;
  supportText?: ReactNode;
  isInvalid?: boolean;
  isReadOnly?: boolean;
  isDisabled?: boolean;
  isRequired?: boolean;
  className?: string;
  "aria-label"?: string;
  "aria-labelledby"?: string;
};

/**
 * A field whose value is set in a popover: the trigger shows the value like
 * a select, and the popover holds the fields that compose it, under an
 * optional title and above an optional footer.
 */
export function FieldPopover({
  value,
  children,
  title,
  footer,
  label,
  placeholder,
  supportText,
  isInvalid,
  isReadOnly,
  isDisabled,
  isRequired,
  className,
  ...props
}: FieldPopoverProps) {
  return (
    <div
      className={clsx("group flex flex-col gap-1", className)}
      data-invalid={isInvalid || undefined}
    >
      {label && <FieldLabel isRequired={isRequired}>{label}</FieldLabel>}

      <DialogTrigger>
        <Button
          {...props}
          isDisabled={isReadOnly || isDisabled}
          className={clsx(
            "flex h-9.5 min-w-0 items-center justify-between gap-2 rounded-lg border px-3 outline-0 -outline-offset-2",
            "border-primary bg-primary hover:bg-primary-hover",
            "group-data-invalid:border-error",
            "data-focus-visible:outline-focus data-pressed:border-accent-heavy data-focus-visible:outline-2",
            isReadOnly
              ? "bg-tertiary border-tertiary hover:bg-tertiary"
              : "data-disabled:border-tertiary data-disabled:bg-disable data-disabled:text-disable data-disabled:cursor-not-allowed"
          )}
        >
          <Typography
            as="span"
            variant="body"
            size="sm"
            className={clsx(
              "min-w-0 truncate text-left",
              value ? "text-primary" : "text-placeholder"
            )}
          >
            {value || placeholder}
          </Typography>
          <Icon
            type="caret-down"
            size="md"
            className={clsx(
              "text-tertiary shrink-0",
              isReadOnly && "invisible"
            )}
            aria-hidden
          />
        </Button>

        <AriaPopover
          offset={4}
          containerPadding={8}
          placement="bottom start"
          className="border-primary bg-primary rounded-lg border shadow-lg"
        >
          <Dialog
            aria-label={title ? undefined : (label ?? props["aria-label"])}
            className="outline-none"
          >
            {title && (
              <Heading
                slot="title"
                className="border-secondary border-b px-4 py-3"
              >
                <Typography as="span" size="sm" weight="semibold">
                  {title}
                </Typography>
              </Heading>
            )}
            <div className="p-4">{children}</div>
            {footer && (
              <div className="border-secondary bg-secondary rounded-b-lg border-t px-4 py-3">
                {footer}
              </div>
            )}
          </Dialog>
        </AriaPopover>
      </DialogTrigger>

      {supportText && <FieldSupportText>{supportText}</FieldSupportText>}
    </div>
  );
}
