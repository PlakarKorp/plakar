import {
  FieldButton,
  Icon,
  Link,
  SegmentedControl,
  SegmentedControlProps,
  Spinner,
  Typography,
} from "@plakar-ui/ui/components";
import { ReactNode, useEffect, useRef } from "react";
import { Input, SearchField } from "react-aria-components";

import { useDebouncedSearch } from "../../hooks/use-debounced-search";

const SEARCH_SHORTCUT = "/";

function ownsTheKeyboard(target: EventTarget | null) {
  return (
    target instanceof HTMLElement &&
    (target.isContentEditable ||
      ["INPUT", "TEXTAREA", "SELECT"].includes(target.tagName) ||
      !!target.closest('[role="dialog"], [role="listbox"], [role="menu"]'))
  );
}

function FilterPanelRoot({ children }: { children: ReactNode }) {
  return (
    <div className="border-secondary bg-primary w-full rounded-xl border shadow-xs">
      {children}
    </div>
  );
}

type FilterPanelSearchProps = {
  value: string;
  placeholder: string;
  "aria-label": string;
  onSearch: (value: string) => void;
  /** Shows a spinner in place of the search icon while the results load. */
  isLoading?: boolean;
};

function FilterPanelSearch({
  value,
  placeholder,
  "aria-label": ariaLabel,
  onSearch,
  isLoading = false,
}: FilterPanelSearchProps) {
  const search = useDebouncedSearch(value, onSearch);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const focusOnShortcut = (event: KeyboardEvent) => {
      if (
        event.key !== SEARCH_SHORTCUT ||
        event.metaKey ||
        event.ctrlKey ||
        event.altKey ||
        ownsTheKeyboard(event.target)
      ) {
        return;
      }
      event.preventDefault();
      inputRef.current?.focus();
    };

    document.addEventListener("keydown", focusOnShortcut);
    return () => document.removeEventListener("keydown", focusOnShortcut);
  }, []);

  return (
    <SearchField
      aria-label={ariaLabel}
      value={search.value}
      onChange={search.change}
      onClear={search.clear}
      className="border-secondary flex items-center gap-3 border-b px-4 last:rounded-b-xl last:border-b-0 has-data-focused:shadow-[inset_0_-2px_0_var(--border-color-accent)]"
    >
      {isLoading ? (
        <Spinner size="md" inheritColor className="text-tertiary" />
      ) : (
        <Icon type="magnifying-glass" size="md" className="text-tertiary" />
      )}
      <Input
        ref={inputRef}
        placeholder={placeholder}
        className="placeholder:text-tertiary text-primary h-12 min-w-0 flex-1 bg-transparent font-sans text-sm outline-none [&::-webkit-search-cancel-button]:hidden"
      />
      {search.value ? (
        <FieldButton slot="close" icon="close" aria-label="Clear search" />
      ) : (
        <kbd
          aria-hidden="true"
          className="bg-tertiary border-secondary text-tertiary hidden shrink-0 rounded-md border border-b-2 px-1.5 py-0.5 font-sans text-xs font-semibold sm:block"
        >
          {SEARCH_SHORTCUT}
        </kbd>
      )}
    </SearchField>
  );
}

function FilterPanelGroups({ children }: { children: ReactNode }) {
  return (
    <div className="border-secondary flex flex-wrap items-end gap-x-8 gap-y-3 border-b px-4 pt-3 pb-4 last:rounded-b-xl last:border-b-0">
      {children}
    </div>
  );
}

function FilterPanelGroup({
  label,
  children,
}: {
  label: string;
  children: ReactNode;
}) {
  return (
    <div className="flex min-w-0 flex-col gap-1.5 max-md:w-full">
      <Typography
        as="span"
        size="xs"
        weight="medium"
        className="text-tertiary tracking-widest uppercase"
      >
        {label}
      </Typography>
      {children}
    </div>
  );
}

type FilterPanelFooterProps = {
  shown: number | undefined;
  total: number | undefined;
  noun: string;
  onClear?: () => void;
  sort?: Pick<SegmentedControlProps, "items" | "selectedId" | "onSelect">;
};

function FilterPanelFooter({
  shown,
  total,
  noun,
  onClear,
  sort,
}: FilterPanelFooterProps) {
  return (
    <div className="bg-secondary border-secondary flex flex-wrap items-center justify-between gap-3 border-b px-4 py-2 last:rounded-b-xl last:border-b-0">
      <span className="flex items-center gap-3">
        <Typography
          as="span"
          size="xs"
          className="text-tertiary"
          aria-live="polite"
        >
          {shown ?? "–"} / {total ?? "–"} {noun}
        </Typography>
        {onClear && (
          <Link onPress={onClear} className="flex">
            <Typography as="span" size="xs" weight="medium">
              Clear filters
            </Typography>
          </Link>
        )}
      </span>
      {sort && (
        <span className="flex items-center gap-2">
          <Typography as="span" size="xs" className="text-tertiary">
            Sort by
          </Typography>
          <SegmentedControl {...sort} variant="accent" ariaLabel="Sort by" />
        </span>
      )}
    </div>
  );
}

export const FilterPanel = Object.assign(FilterPanelRoot, {
  Search: FilterPanelSearch,
  Groups: FilterPanelGroups,
  Group: FilterPanelGroup,
  Footer: FilterPanelFooter,
});
