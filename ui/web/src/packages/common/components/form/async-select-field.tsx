import { Select, SelectProps, SelectOption } from "@plakar-ui/ui/components";
import { useEffect } from "react";
import { Collection } from "react-aria-components";
import { useAsyncList } from "react-stately";

import { useFieldContext } from "../../hooks/form-context";
import { useFormReadOnly } from "../../hooks/use-form-read-only";

type AsyncSelectFieldProps = {
  list: Pick<
    ReturnType<typeof useAsyncList<SelectOption>>,
    "items" | "loadingState" | "filterText" | "setFilterText" | "loadMore"
  >;
  selectedOption?: SelectOption;
  autoSelectSingleOption?: boolean;
  search?: boolean;
} & Omit<
  SelectProps,
  | "value"
  | "onChange"
  | "onBlur"
  | "onOpenChange"
  | "isInvalid"
  | "selectedFallbackOption"
>;

export function AsyncSelectField({
  list,
  selectedOption,
  supportText,
  autoSelectSingleOption = false,
  search = true,
  isDisabled,
  isPending,
  ...props
}: AsyncSelectFieldProps) {
  const field = useFieldContext<string>();
  const isReadOnly = useFormReadOnly();

  const isInitialLoading =
    list.loadingState === "loading" && list.items.length === 0;

  // Set when there is only one option in the list, and no filter is applied.
  const singleOptionId =
    list.filterText === "" && list.items.length === 1
      ? list.items[0].id
      : undefined;

  // Auto-select the only option.
  useEffect(() => {
    if (!autoSelectSingleOption) {
      return;
    }
    // Wait for the list to finish loading before auto-selecting.
    if (list.loadingState !== "idle") {
      return;
    }
    if (!singleOptionId) {
      return;
    }
    if (!field.state.value) {
      field.setValue(singleOptionId, { dontUpdateMeta: true });
    }
  }, [
    singleOptionId,
    field,
    field.state.value,
    list.loadingState,
    autoSelectSingleOption,
  ]);

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
      value={field.state.value ?? null}
      onChange={(key) => field.handleChange(key ? String(key) : "")}
      onOpenChange={(isOpen: boolean) => {
        if (!isOpen && list.filterText) list.setFilterText("");
      }}
      onBlur={field.handleBlur}
      isInvalid={field.state.meta.errors.length > 0}
      supportText={field.state.meta.errors[0] || supportText}
      selectedFallbackOption={selectedOption}
      isPending={isInitialLoading || isPending}
      // A field with a known selectedOption has something worth showing and
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
      isReadOnly={isReadOnly || props.isReadOnly}
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
