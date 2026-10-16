import { FieldButton, TextInput } from "@plakar-ui/ui/components";
import { useDebouncer } from "@tanstack/react-pacer/debouncer";
import clsx from "clsx";
import { ReactNode, useEffect, useState } from "react";
import { Key } from "react-aria-components";

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
  const [localSearch, setLocalSearch] = useState(search);

  // Sync local state when the URL changes (eg. back/forward navigation).
  useEffect(() => {
    setLocalSearch(search);
  }, [search]);

  const searchDebouncer = useDebouncer((value: string) => onSearch(value), {
    wait: 300,
  });

  const handleFilenameChange = (key: Key | null) => {
    const newValue = key ? String(key) : "";
    setLocalSearch(newValue);
    searchDebouncer.maybeExecute(newValue);
  };

  const handleClear = () => {
    setLocalSearch("");
    // Instant clear of search without debounce
    searchDebouncer.cancel();
    onSearch("");
  };

  return (
    <div className="flex w-full flex-wrap items-center gap-2">
      <div
        className={clsx("order-1", isFullWidth ? "flex-1" : "max-sm:flex-1")}
      >
        <TextInput
          icon="magnifying-glass"
          placeholder={searchPlaceholder}
          className="w-full sm:w-64"
          value={localSearch}
          onChange={handleFilenameChange}
          aria-label="Search by name"
          suffix={
            localSearch ? (
              <FieldButton
                icon="close"
                aria-label="Clear text"
                onPress={handleClear}
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
