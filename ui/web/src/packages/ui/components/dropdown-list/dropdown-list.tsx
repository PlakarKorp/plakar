import clsx from "clsx";
import { ReactNode, useState } from "react";
import {
  Autocomplete,
  Dialog,
  Popover,
  PopoverProps,
  useFilter,
} from "react-aria-components";

import { EmptyList } from "../empty-list/empty-list";
import { FieldSearch } from "../field/field-search";
import { Icon } from "../icon/icon";
import { IconType } from "../icon/icon.constants";
import { SelectListBoxItem } from "../select/select-item";
import { SelectListBox } from "../select/select-list-box";
import { SelectLoading } from "../select/select-loading";
import { Spinner } from "../spinner/spinner";
import { Tag } from "../tag/tag";
import { TextHighlight } from "../text-highlight/text-highlight";
import { Typography } from "../typography/typography";

export type DropdownListItem = {
  id: string;
  name: string;
  description?: ReactNode;
  icon?: IconType;
  tag?: ReactNode;
  action?: { label: string; icon: IconType; isLoading?: boolean };
  isDisabled?: boolean;
};

export type DropdownListProps = {
  label: string;
  items: DropdownListItem[];
  selectedId?: string | null;
  onSelect: (id: string) => void;
  searchPlaceholder?: string;
  emptyTitle?: string;
  emptyDescription?: string;
  isLoading?: boolean;
  onLoadMore?: () => void;
  inputValue?: string;
  onInputChange?: (value: string) => void;
};

export function DropdownList({
  label,
  items,
  selectedId,
  onSelect,
  searchPlaceholder,
  emptyTitle = "Nothing here yet",
  emptyDescription = "This list is empty for now.",
  isLoading = false,
  onLoadMore,
  inputValue,
  onInputChange,
}: DropdownListProps) {
  const { contains } = useFilter({ sensitivity: "base" });
  const [searchText, setSearchText] = useState("");
  const search = inputValue ?? searchText;

  if (items.length === 0 && !search && !isLoading) {
    return <EmptyList title={emptyTitle} description={emptyDescription} />;
  }

  return (
    <Autocomplete
      filter={onInputChange ? undefined : contains}
      inputValue={search}
      onInputChange={(value) => {
        setSearchText(value);
        onInputChange?.(value);
      }}
    >
      <FieldSearch label={`Search ${label}`} placeholder={searchPlaceholder} />

      <SelectListBox
        aria-label={label}
        // Fills the popover, which React Aria sizes to the space on screen.
        className="flex min-h-0 flex-col gap-1 overflow-auto p-1 outline-0"
        selectionMode="single"
        selectedKeys={selectedId ? [selectedId] : []}
        onSelectionChange={(keys) => {
          const [key] = keys === "all" ? [] : [...keys];
          if (key !== undefined) {
            onSelect(String(key));
          }
        }}
        renderEmptyState={() =>
          !isLoading && (
            <Typography
              as="div"
              variant="body"
              size="sm"
              className="text-tertiary p-2 italic"
            >
              No results found
            </Typography>
          )
        }
      >
        {items.map((item) => (
          <SelectListBoxItem
            key={item.id}
            id={item.id}
            name={item.name}
            isDisabled={item.isDisabled}
            className={clsx(
              item.isDisabled ? "cursor-default" : "cursor-pointer"
            )}
            item={<DropdownListRow item={item} search={search} />}
          />
        ))}

        {(onLoadMore || isLoading) && (
          <SelectLoading onLoadMore={onLoadMore} isLoading={isLoading} />
        )}
      </SelectListBox>
    </Autocomplete>
  );
}

type DropdownListPopoverProps = Pick<PopoverProps, "placement"> & {
  label: string;
  children: (close: () => void) => ReactNode;
};

export function DropdownListPopover({
  label,
  placement = "bottom start",
  children,
}: DropdownListPopoverProps) {
  return (
    <Popover
      offset={4}
      placement={placement}
      className="border-primary bg-primary flex w-96 max-w-[calc(100vw-2rem)] flex-col rounded-lg border shadow-lg"
    >
      <Dialog aria-label={label} className="flex min-h-0 flex-col outline-none">
        {({ close }) => children(close)}
      </Dialog>
    </Popover>
  );
}

type DropdownListRowProps = {
  item: DropdownListItem;
  search: string;
};

function DropdownListRow({ item, search }: DropdownListRowProps) {
  return (
    <div className="flex w-full items-center gap-2">
      {item.icon && (
        <Icon type={item.icon} size="md" className="text-tertiary shrink-0" />
      )}

      <div className="min-w-0 flex-1">
        <Typography as="div" size="sm" className="text-primary truncate">
          {item.action && (
            <span className="text-accent mr-1 inline-flex items-center gap-1 align-bottom font-medium">
              {item.action.isLoading ? (
                <Spinner size="sm" inheritColor />
              ) : (
                <Icon type={item.action.icon} size="sm" />
              )}
              {item.action.label}
            </span>
          )}
          <TextHighlight text={item.name} search={search} />
        </Typography>

        {item.description !== null && item.description !== undefined && (
          <Typography as="div" size="xs" className="text-tertiary truncate">
            {item.description}
          </Typography>
        )}
      </div>

      {item.tag !== null && item.tag !== undefined && (
        <Tag radius="sqrt" size="sm" variant="default_borderless">
          {item.tag}
        </Tag>
      )}
    </div>
  );
}
