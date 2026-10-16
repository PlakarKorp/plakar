import clsx from "clsx";
import { ReactNode } from "react";
import { Button, DialogTrigger } from "react-aria-components";

import { getTabTriggerClassName, TabVariant } from "./tab-trigger";
import {
  DropdownList,
  DropdownListPopover,
  DropdownListProps,
} from "../dropdown-list/dropdown-list";
import { Icon } from "../icon/icon";
import { IconType } from "../icon/icon.constants";
import { Tag } from "../tag/tag";
import { Typography } from "../typography/typography";

export type TabDropdownProps = DropdownListProps & {
  variant?: TabVariant;
  icon?: IconType;
  isSelected?: boolean;
  tag?: ReactNode;
  isDisabled?: boolean;
  className?: string;
};

/**
 * Tab-shaped dropdown: looks like a tab, opens a searchable list. Meant for a
 * tab whose options are dynamic, too many, or paginated.
 *
 * It must be rendered as a *sibling* of `Tab`, never inside it — React Aria
 * builds the tab collection by rendering the subtree of `Tabs`, which
 * swallows the items of any nested collection component.
 */
export function TabDropdown({
  variant = "underlined",
  label,
  icon,
  items,
  selectedId,
  onSelect,
  isSelected,
  tag,
  isDisabled,
  onInputChange,
  className,
  ...listProps
}: TabDropdownProps) {
  const selectedItem = items.find((item) => item.id === selectedId) ?? null;
  const isActive = isSelected ?? selectedItem !== null;

  const activeItem = isActive ? selectedItem : null;
  const triggerIcon = activeItem?.icon ?? icon;

  return (
    <DialogTrigger
      onOpenChange={(isOpen) => {
        if (!isOpen) {
          onInputChange?.("");
        }
      }}
    >
      <Button
        isDisabled={isDisabled}
        className={clsx(
          getTabTriggerClassName({ variant, isSelected: isActive, isDisabled }),
          className
        )}
      >
        {triggerIcon && <Icon type={triggerIcon} size="md" />}

        <Typography
          as="span"
          size="sm"
          weight={variant === "pill" ? "medium" : "normal"}
        >
          {activeItem ? activeItem.name : label}
        </Typography>

        {tag !== null && tag !== undefined && (
          <Tag
            radius="sqrt"
            size="sm"
            variant={isDisabled ? "disabled" : "default_borderless"}
          >
            {tag}
          </Tag>
        )}

        <Icon type="caret-down" size="md" />

        {variant === "underlined" && isActive && (
          <span
            aria-hidden="true"
            className="border-accent-heavy absolute -bottom-0.5 left-0 h-1 w-full border-b-2"
          />
        )}
      </Button>

      <DropdownListPopover label={label}>
        {(close) => (
          <DropdownList
            {...listProps}
            label={label}
            items={items}
            selectedId={selectedId}
            onInputChange={onInputChange}
            onSelect={(id) => {
              onSelect(id);
              close();
            }}
          />
        )}
      </DropdownListPopover>
    </DialogTrigger>
  );
}
