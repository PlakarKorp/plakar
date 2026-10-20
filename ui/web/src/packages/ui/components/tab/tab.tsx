import clsx from "clsx";
import { ReactNode } from "react";
import {
  Tabs as RATabs,
  Tab as RATab,
  TabList as RATabList,
  TabPanel as RATabPanel,
  Collection as RACollection,
  Key,
  LinkProps,
  SelectionIndicator,
} from "react-aria-components";

import { Icon, IconType, Tag, Tooltip, Typography } from "..";
import { TabDropdown } from "./tab-dropdown";
import { getTabTriggerClassName, TabVariant } from "./tab-trigger";

export type { TabDropdownProps } from "./tab-dropdown";

const NO_TAB_SELECTED = "__none__";

export type TabItem = Pick<LinkProps, "href"> & {
  id: string;
  title: string;
  icon?: IconType;
  tag?: ReactNode;
  content?: React.ReactNode;
  isDisabled?: boolean;
  tooltip?: string;
};

// Either set `selected` *and* `link` in items, or omit both.
type TabProps<T extends readonly TabItem[]> = {
  variant: TabVariant;
  ariaLabel: string;
  selected?: string | null;
  items: T;
  className?: string;
  fullWidth?: boolean;
  onSelectionChange?: (key: Key) => void;
  action?: ReactNode;
};

function TabRoot<const T extends readonly TabItem[]>({
  variant,
  ariaLabel,
  items,
  selected,
  className,
  fullWidth,
  onSelectionChange,
  action,
}: TabProps<T>) {
  return (
    <RATabs
      selectedKey={selected === null ? NO_TAB_SELECTED : selected}
      onSelectionChange={onSelectionChange}
      className={clsx("max-w-full min-w-0", className)}
    >
      <div className="flex items-center gap-2">
        <RATabList
          aria-label={ariaLabel}
          items={items}
          className={clsx("scrollbar-hide flex min-w-0 overflow-auto", {
            "max-w-fit gap-2": variant === "underlined",
            "border-secondary bg-tertiary gap-1 rounded-lg border p-0.5":
              variant === "pill",
            "flex-1": fullWidth,
          })}
        >
          {items.map((item) => (
            <RATab
              key={item.id}
              id={item.id}
              isDisabled={item.isDisabled}
              className={clsx(
                "flex-1",
                getTabTriggerClassName({
                  variant,
                  isSelected: selected === item.id,
                  isDisabled: item.isDisabled,
                })
              )}
              {...(item.href ? { href: item.href } : {})}
            >
              <TabLabel item={item} variant={variant} />

              {variant === "underlined" && (
                <SelectionIndicator className="border-accent-heavy absolute -bottom-0.5 left-0 h-1 w-full border-b-2 motion-safe:transition-[translate,width,height]" />
              )}
            </RATab>
          ))}
        </RATabList>
        {action && <div className="shrink-0">{action}</div>}
      </div>
      {items.some(({ content }) => content) && (
        <RACollection items={items}>
          {(item) => (
            <RATabPanel className="flex h-full min-h-0 flex-col">
              {item.content && (
                <div className="border-secondary flex min-h-0 flex-1 flex-col border-t">
                  {item.content}
                </div>
              )}
            </RATabPanel>
          )}
        </RACollection>
      )}
    </RATabs>
  );
}

function TabLabel({ item, variant }: { item: TabItem; variant: TabVariant }) {
  const label = (
    <>
      {item.icon && <Icon type={item.icon} size="md" />}

      <Typography
        as="span"
        size="sm"
        weight={variant === "pill" ? "medium" : "normal"}
      >
        {item.title}
      </Typography>

      {item.tag !== null && item.tag !== undefined && (
        <Tag
          radius="sqrt"
          size="sm"
          variant={item.isDisabled ? "disabled" : "default_borderless"}
        >
          {item.tag}
        </Tag>
      )}
    </>
  );

  return item.tooltip ? (
    <Tooltip label={item.tooltip}>
      <span tabIndex={-1} className="flex items-center gap-2 outline-0">
        {label}
      </span>
    </Tooltip>
  ) : (
    label
  );
}

export const Tab = Object.assign(TabRoot, {
  Dropdown: TabDropdown,
});
