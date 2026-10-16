import clsx from "clsx";
import { ComponentProps, ReactNode } from "react";
import { Switch as RASwitch } from "react-aria-components";

import { FieldLabel } from "../field/field-label";
import { FieldSupportText } from "../field/field-support-text";

export type ToggleProps = ComponentProps<typeof RASwitch> &
  (
    | {
        title?: ReactNode;
        description?: never;
        beforeLabelAction?: ReactNode;
        labelAction?: ReactNode;
      }
    | {
        title: ReactNode;
        description?: ReactNode;
        beforeLabelAction?: ReactNode;
        labelAction?: ReactNode;
      }
  );

export function Toggle({
  title,
  description,
  beforeLabelAction,
  labelAction,
  ...props
}: ToggleProps) {
  return (
    <div className="flex flex-col [contain:layout]">
      <RASwitch
        {...props}
        className={clsx(
          "group flex w-fit cursor-pointer gap-2 data-disabled:cursor-default",
          props.className
        )}
      >
        {({ isSelected }) => (
          <>
            <div
              className={clsx(
                "relative inline-flex h-6 w-10 shrink-0 rounded-full border-3 border-transparent transition-colors duration-200 ease-in-out",
                "group-data-selected:bg-accent group-data-selected:hover:bg-accent-hover bg-zinc-200 group-data-disabled:bg-zinc-300 dark:bg-zinc-600 group-data-disabled:dark:bg-zinc-800", // hardcoded colors because DS colors don't have enough contrast
                "group-data-focus-visible:outline-focus group-data-focus-visible:outline-2 group-data-focus-visible:outline-offset-2"
              )}
            >
              <div
                className={clsx(
                  "pointer-events-none inline-block size-4.5 transform rounded-full bg-white shadow-sm transition duration-200 ease-in-out group-data-disabled:bg-zinc-200 group-data-disabled:dark:bg-zinc-700",
                  isSelected && "translate-x-4"
                )}
              />
            </div>
            {title && (
              <FieldLabel
                as="span"
                className="leading-6"
                beforeAction={beforeLabelAction}
                action={labelAction}
              >
                {title}
              </FieldLabel>
            )}
          </>
        )}
      </RASwitch>
      {description && (
        <FieldSupportText className="ml-12">{description}</FieldSupportText>
      )}
    </div>
  );
}
