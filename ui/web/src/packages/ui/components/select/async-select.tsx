import { Collection } from "react-aria-components";
import { useAsyncList } from "react-stately";

import { Select, SelectOption, SelectProps } from "./select";

export type AsyncSelectProps = {
  list: Pick<
    ReturnType<typeof useAsyncList<SelectOption>>,
    "items" | "loadingState" | "filterText" | "setFilterText" | "loadMore"
  >;
  selectedOption?: SelectOption;
  search?: boolean;
} & Omit<SelectProps, "selectedFallbackOption" | "onOpenChange">;

/**
 * A select whose options are loaded page by page as the list is scrolled,
 * and optionally searched.
 */
export function AsyncSelect({
  list,
  selectedOption,
  search = true,
  isDisabled,
  isPending,
  ...props
}: AsyncSelectProps) {
  const isInitialLoading =
    list.loadingState === "loading" && list.items.length === 0;

  const listboxContent = (
    <>
      <Collection items={list.items}>
        {(item) => <Select.Item key={item.id} {...item} />}
      </Collection>
      <Select.Loading
        onLoadMore={list.loadMore}
        isLoading={list.loadingState === "loadingMore"}
      />
    </>
  );

  return (
    <Select
      onOpenChange={(isOpen: boolean) => {
        if (!isOpen && list.filterText) list.setFilterText("");
      }}
      selectedFallbackOption={selectedOption}
      isPending={isInitialLoading || isPending}
      // A select with a known selectedOption has something worth showing and
      // interacting with even before the list has data -- opening it just
      // means seeing an empty/loading list rather than nothing usable at
      // all, so it shouldn't be forced disabled on that account.
      isDisabled={
        isDisabled ||
        (isInitialLoading && !selectedOption) ||
        (list.items.length === 0 &&
          !selectedOption &&
          !props.allowsEmptyCollection &&
          !list.filterText)
      }
      {...props}
    >
      {search ? (
        <Select.SearchableListBox
          inputValue={list.filterText}
          onInputChange={list.setFilterText}
        >
          {listboxContent}
        </Select.SearchableListBox>
      ) : (
        <Select.ListBox>{listboxContent}</Select.ListBox>
      )}
    </Select>
  );
}
