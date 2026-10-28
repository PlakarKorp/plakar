import clsx from "clsx";
import { ReactNode, useContext, useLayoutEffect, useRef } from "react";
import {
  Radio as AriaRadio,
  RadioProps as AriaRadioProps,
  RadioGroup as AriaRadioGroup,
  RadioGroupProps as AriaRadioGroupProps,
  RadioGroupStateContext,
} from "react-aria-components";

import { FieldLabel } from "../field/field-label";
import { FieldSupportText } from "../field/field-support-text";
import { Icon } from "../icon/icon";
import { Typography } from "../typography/typography";

export type RadioGroupProps = Omit<AriaRadioGroupProps, "children"> & {
  children?: ReactNode;
  label?: string;
  supportText?: string;
  spacing?: 1 | 2 | 4;
  layout?: "stack" | "grid";
};

export function RadioGroup({
  label,
  children,
  supportText,
  spacing = 2,
  layout = "stack",
  className,
  ...props
}: RadioGroupProps) {
  const gap = {
    1: "gap-1",
    2: "gap-2",
    4: "gap-4",
  }[spacing];

  return (
    <AriaRadioGroup
      {...props}
      className={clsx("group flex flex-col", gap, className)}
    >
      {label && <FieldLabel isRequired={props.isRequired}>{label}</FieldLabel>}
      {layout === "grid" ? (
        <div
          className={clsx(
            "grid grid-cols-[repeat(auto-fit,minmax(min(100%,12rem),1fr))]",
            gap
          )}
        >
          {children}
        </div>
      ) : (
        <>{children}</>
      )}
      {supportText && <FieldSupportText>{supportText}</FieldSupportText>}
    </AriaRadioGroup>
  );
}

type RadioProps = Omit<AriaRadioProps, "children"> & {
  children?: ReactNode;
  description?: string;
};

export function Radio({
  children,
  description,
  className,
  ...props
}: RadioProps) {
  return (
    <AriaRadio
      {...props}
      className={clsx(
        "text-secondary relative flex w-fit gap-2 text-sm",
        description ? "items-start" : "items-center",
        // Base circle
        "before:border-primary before:bg-primary before:box-border before:block before:h-5 before:w-5 before:shrink-0 before:rounded-full before:border before:content-['']",
        // Selected state indicator
        "after:bg-accent after:absolute after:top-0.75 after:left-0.75 after:h-3.5 after:w-3.5 after:scale-0 after:rounded-full after:transition-transform after:duration-150 after:ease-in-out after:content-['']",
        // States
        "data-disabled:text-tertiary data-disabled:before:border-primary data-disabled:cursor-not-allowed",
        "group-data-invalid:before:border-error",
        "data-pressed:before:border-accent-heavy",
        "data-selected:before:border-accent data-selected:text-primary data-selected:font-medium data-selected:after:scale-100",
        "data-focus-visible:before:outline-focus data-focus-visible:before:outline-2 data-focus-visible:before:-outline-offset-1",
        className
      )}
    >
      {description ? (
        <span className="flex flex-col gap-0.5">
          <span>{children}</span>
          <Typography as="span" size="xs" className="text-tertiary">
            {description}
          </Typography>
        </span>
      ) : (
        children
      )}
    </AriaRadio>
  );
}

export type CardRadioProps = Omit<AriaRadioProps, "children"> & {
  label: string;
  description?: string;
};

export function CardRadio({
  label,
  description,
  className,
  ...props
}: CardRadioProps) {
  return (
    <AriaRadio
      {...props}
      className={clsx(
        "group/card-radio",
        "flex items-center gap-4 rounded-lg border p-4 -outline-offset-1",
        "border-primary bg-primary text-primary cursor-pointer",
        // States
        "data-disabled:text-tertiary data-disabled:cursor-not-allowed",
        "group-data-invalid:border-error",
        "data-selected:text-primary data-selected:outline-focus data-selected:bg-accent-tertiary data-selected:outline-2",
        className
      )}
    >
      {({ isSelected }) => (
        <>
          <div
            className={clsx(
              "flex h-10 w-10 shrink-0 items-center justify-center rounded-lg",
              "text-primary bg-tertiary text-lg font-medium",
              // States
              "group-data-disabled/card-radio:text-tertiary",
              "group-data-selected/card-radio:bg-accent group-data-selected/card-radio:text-on-accent dark:group-data-selected/card-radio:text-primary"
            )}
          >
            {label.charAt(0).toUpperCase()}
          </div>
          <div className="flex flex-1 flex-col">
            <span>{label}</span>
            {description && (
              <span className="text-tertiary text-sm">{description}</span>
            )}
          </div>
          {isSelected && (
            <Icon type="check" size="md" className="text-accent" />
          )}
        </>
      )}
    </AriaRadio>
  );
}

export type ExpandableCardRadioProps = Omit<AriaRadioProps, "children"> & {
  label: string;
  children?: ReactNode;
  suffix?: ReactNode;
  expandedContent?: ReactNode;
  scrollsIntoViewOnSelect?: boolean;
};

/**
 * A radio card that says a little in the list and everything once chosen, for
 * options that each need an explanation. Both halves are caller-supplied, so
 * the card carries no assumption about what an option is.
 */
export function ExpandableCardRadio({
  label,
  children,
  suffix,
  expandedContent,
  scrollsIntoViewOnSelect = false,
  className,
  ...props
}: ExpandableCardRadioProps) {
  const cardRef = useRef<HTMLLabelElement>(null);
  const state = useContext(RadioGroupStateContext);
  const isSelected = state?.selectedValue === props.value;

  // After the layout that revealed the expanded content, so the card is
  // measured at its final height.
  useLayoutEffect(() => {
    const card = cardRef.current;
    const container = card?.parentElement;

    if (!scrollsIntoViewOnSelect || !isSelected || !card || !container) {
      return;
    }

    // Not `scrollIntoView`: it walks up every scrollable ancestor, so it would
    // scroll the page as well as the list.
    const paddingTop = parseFloat(getComputedStyle(container).paddingTop) || 0;
    const distance =
      card.getBoundingClientRect().top -
      container.getBoundingClientRect().top -
      paddingTop;

    container.scrollTop += distance;
  }, [isSelected, scrollsIntoViewOnSelect]);

  return (
    <AriaRadio
      {...props}
      ref={cardRef}
      className={clsx(
        "group/card-radio",
        "flex flex-col gap-2 rounded-lg border p-3 -outline-offset-1",
        "border-primary bg-primary text-primary cursor-pointer",
        "hover:bg-primary-hover",
        // States
        "data-disabled:text-tertiary data-disabled:cursor-not-allowed",
        "group-data-invalid:border-error",
        "data-selected:text-primary data-selected:outline-focus data-selected:bg-accent-tertiary data-selected:outline-2",
        className
      )}
    >
      <div className="flex items-center gap-2">
        <div
          className={clsx(
            "h-2 w-2 shrink-0 rounded-full",
            "bg-tertiary group-hover/card-radio:bg-accent-secondary",
            // States
            "group-data-selected/card-radio:bg-accent"
          )}
        />
        <Typography size="sm" weight="medium" className="flex-1 truncate">
          {label}
        </Typography>
        {isSelected ? (
          <Icon type="check" size="md" className="text-accent shrink-0" />
        ) : (
          suffix
        )}
      </div>

      {isSelected && expandedContent ? (
        <div className="pl-4">{expandedContent}</div>
      ) : (
        children && <div className="min-w-0 pl-4">{children}</div>
      )}
    </AriaRadio>
  );
}

export function CompactCardRadio({
  label,
  description,
  className,
  ...props
}: CardRadioProps) {
  return (
    <AriaRadio
      {...props}
      className={clsx(
        "group/card-radio",
        "flex items-center gap-4 rounded-lg border px-4 py-3 -outline-offset-1",
        "border-primary bg-primary text-primary cursor-pointer",
        "hover:bg-primary-hover",
        // States
        "data-disabled:text-tertiary data-disabled:cursor-not-allowed",
        "group-data-invalid:border-error",
        "data-selected:text-primary data-selected:outline-focus data-selected:bg-accent-tertiary data-selected:outline-2",
        className
      )}
    >
      {({ isSelected }) => (
        <>
          <div
            className={clsx(
              "h-2 w-2 rounded-full",
              "bg-tertiary group-hover/card-radio:bg-accent-secondary",
              // States
              "group-data-selected/card-radio:bg-accent"
            )}
          />
          <div className="flex flex-1 items-center justify-between gap-2">
            <Typography size="sm" weight="medium">
              {label}
            </Typography>
            {description && (
              <Typography size="xs" className="text-tertiary">
                {description}
              </Typography>
            )}
          </div>
          {isSelected && (
            <Icon type="check" size="md" className="text-accent" />
          )}
        </>
      )}
    </AriaRadio>
  );
}
