import clsx from "clsx";
import { PropsWithChildren, useLayoutEffect, useRef, useState } from "react";
import {
  Autocomplete,
  AutocompleteProps,
  ListBox,
  Popover,
  Select as AriaSelect,
  SelectProps as AriaSelectProps,
  SelectValue,
} from "react-aria-components";

import { IconButton, IconType, RemovableTags, Typography } from "..";
import { FieldLabel } from "../field/field-label";
import { FieldSearch } from "../field/field-search";
import { FieldSupportText } from "../field/field-support-text";
import { SelectListBoxItem } from "../select/select-item";
import { SelectListBox } from "../select/select-list-box";

export type MultiSelectOption = {
  id: string;
  name: string;
  icon?: IconType;
  onPress?: () => void;
};

export type MultiSelectProps = PropsWithChildren<
  AriaSelectProps<MultiSelectOption, "multiple">
> & {
  label?: string;
  supportText?: string;
  isPending?: boolean;
};

function MultiSelectRoot({
  label,
  supportText,
  className,
  children,
  isPending = false,
  ...props
}: MultiSelectProps) {
  const triggerRef = useRef<HTMLDivElement | null>(null);
  const [triggerWidth, setTriggerWidth] = useState<number>();

  useLayoutEffect(() => {
    if (!triggerRef.current) return;

    const element = triggerRef.current;

    const updateWidth = () => {
      setTriggerWidth(element.offsetWidth);
    };

    updateWidth();

    if (typeof ResizeObserver === "undefined") {
      return;
    }

    const observer = new ResizeObserver(updateWidth);
    observer.observe(element);

    return () => observer.disconnect();
  }, []);

  return (
    <AriaSelect
      {...props}
      className={clsx("group flex flex-col gap-1", className)}
      selectionMode="multiple"
    >
      {label && <FieldLabel isRequired={props.isRequired}>{label}</FieldLabel>}

      <div className="relative flex min-w-fit flex-1" ref={triggerRef}>
        <div
          className={clsx(
            "flex flex-1 items-center justify-between gap-2 rounded-lg px-3 py-1.25 pr-2 outline-0 -outline-offset-2",
            "border-primary bg-primary text-tertiary border",
            "group-data-invalid:border-error",
            "group-data-open:border-accent-heavy",
            "group-data-disabled:border-tertiary group-data-disabled:bg-disable group-data-disabled:text-disable group-data-disabled:cursor-not-allowed"
          )}
        >
          <SelectValue<MultiSelectOption> className="grid min-h-6.5 grid-cols-1 items-center text-left">
            {({ selectedItems, state }) => {
              const items = selectedItems.filter((item) => item !== null);

              return (
                <RemovableTags
                  label="Selected items"
                  onRemove={(keys) => {
                    if (Array.isArray(state.value)) {
                      state.setValue(state.value.filter((k) => !keys.has(k)));
                    }
                  }}
                  disabledKeys={
                    props.isDisabled ? items.map((t) => t.id) : undefined
                  }
                >
                  {items.map(({ id, name, icon }) => (
                    <RemovableTags.Item
                      id={id}
                      key={id}
                      value={name}
                      icon={icon}
                      size="sm"
                    />
                  ))}
                </RemovableTags>
              );
            }}
          </SelectValue>

          <IconButton
            icon="plus"
            size="sm"
            variant="tertiary"
            className="my-0.5"
            isPending={isPending}
          />
        </div>
      </div>

      {supportText && <FieldSupportText>{supportText}</FieldSupportText>}

      <Popover
        triggerRef={triggerRef}
        offset={4}
        containerPadding={8}
        className="border-primary bg-primary flex max-h-80 flex-col rounded-lg border shadow-lg"
        style={triggerWidth ? { width: triggerWidth } : undefined}
      >
        {children}
      </Popover>
    </AriaSelect>
  );
}

type MultiSelectSearchableListBoxProps = AutocompleteProps;

function MultiSelectSearchableListBox({
  children,
  ...props
}: MultiSelectSearchableListBoxProps) {
  return (
    <Autocomplete {...props}>
      <FieldSearch />
      <ListBox
        className="flex flex-1 flex-col gap-1 overflow-auto p-1 outline-0"
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
      >
        {children}
      </ListBox>
    </Autocomplete>
  );
}

export const MultiSelect = Object.assign(MultiSelectRoot, {
  Item: SelectListBoxItem,
  ListBox: SelectListBox,
  SearchableListBox: MultiSelectSearchableListBox,
});
