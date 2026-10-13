import { FieldButton, TextInput } from "@plakar-ui/ui/components";
import clsx from "clsx";
import { ReactNode } from "react";
import { Key } from "react-aria-components";

import { useDebouncedSearch } from "../../hooks/use-debounced-search";

type Props = {
  search: string;
  searchPlaceholder?: string;
  filters?: ReactNode;
  isFullWidth?: boolean;
  onSearch: (search: string) => void;
};

export function TableFilters({
  search,
  searchPlaceholder,
  filters,
  isFullWidth = true,
  onSearch,
}: Props) {
  const localSearch = useDebouncedSearch(search, onSearch);

  return (
    <div className="flex w-full flex-wrap items-center gap-2">
      <div
        className={clsx("order-1", isFullWidth ? "flex-1" : "max-sm:flex-1")}
      >
        <TextInput
          icon="magnifying-glass"
          placeholder={searchPlaceholder}
          className="w-full sm:w-64"
          value={localSearch.value}
          onChange={(key: Key | null) =>
            localSearch.change(key ? String(key) : "")
          }
          aria-label="Search by name"
          suffix={
            localSearch.value ? (
              <FieldButton
                icon="close"
                aria-label="Clear text"
                onPress={localSearch.clear}
                className="-mr-2"
              />
            ) : undefined
          }
        />
      </div>
      {filters}
    </div>
  );
}
