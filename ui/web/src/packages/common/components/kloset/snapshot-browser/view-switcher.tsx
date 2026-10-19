import { Tab } from "@plakar-ui/ui/components";

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
  const items = BROWSE_VIEWS.map((item) => {
    const count = counts?.[item.id];
    return {
      id: item.id,
      title: item.name,
      icon: item.icon,
      tag: count,
      isDisabled: item.id !== "all" && item.id !== "documents" && count === 0,
    };
  });

  return (
    <Tab
      variant="underlined"
      ariaLabel="Browse by kind of content"
      items={items}
      selected={view}
      onSelectionChange={(key) => {
        const next = BROWSE_VIEWS.find((v) => v.id === key);
        if (next) {
          onChange(next.id);
        }
      }}
    />
  );
}
