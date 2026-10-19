import { Icon, Typography } from "@plakar-ui/ui/components";
import clsx from "clsx";
import { Key, ToggleButton, ToggleButtonGroup } from "react-aria-components";

import { BROWSE_VIEWS, BrowseViewId } from "./browse-views";

type ViewSwitcherProps = {
  view: BrowseViewId;
  // Left out while searching: the summary does not count the results.
  counts?: Record<BrowseViewId, number>;
  onChange: (view: BrowseViewId) => void;
};

/**
 * Tabs to browse a directory by kind of content, each with the number of files
 * of that kind it holds. Kinds the directory has none of are disabled, except
 * documents: their count misses PDFs, so it can read zero when there are some.
 */
export function ViewSwitcher({ view, counts, onChange }: ViewSwitcherProps) {
  const handleSelectionChange = (keys: Set<Key>) => {
    const [id] = keys;
    const next = BROWSE_VIEWS.find((v) => v.id === id);
    if (next) {
      onChange(next.id);
    }
  };

  return (
    <ToggleButtonGroup
      aria-label="Browse by kind of content"
      selectionMode="single"
      disallowEmptySelection
      selectedKeys={[view]}
      onSelectionChange={handleSelectionChange}
      className="border-secondary flex flex-row gap-1 overflow-x-auto border-b [scrollbar-width:none]"
    >
      {BROWSE_VIEWS.map((item) => {
        const count = counts?.[item.id];
        const isDisabled =
          item.id !== "all" && item.id !== "documents" && count === 0;
        return (
          <ToggleButton
            key={item.id}
            id={item.id}
            isDisabled={isDisabled}
            className={clsx(
              "-mb-px flex cursor-pointer items-center gap-2 border-b-2 border-transparent px-3 py-2 whitespace-nowrap",
              "text-tertiary data-hovered:text-secondary",
              "data-selected:border-accent data-selected:text-primary",
              "data-focus-visible:outline-focus data-focus-visible:outline-2",
              "data-disabled:text-disable data-disabled:cursor-not-allowed",
              "duration-150 motion-safe:transition-colors"
            )}
          >
            <Icon type={item.icon} size="md" />
            <Typography as="span" size="sm" weight="medium">
              {item.name}
            </Typography>
            {count !== undefined && (
              <Typography
                as="span"
                size="xs"
                className="bg-tertiary text-tertiary rounded-full px-1.5 tabular-nums"
              >
                {count}
              </Typography>
            )}
          </ToggleButton>
        );
      })}
    </ToggleButtonGroup>
  );
}
