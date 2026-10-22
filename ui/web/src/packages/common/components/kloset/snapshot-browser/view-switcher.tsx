import { Tab } from "@plakar-ui/ui/components";

import { BROWSE_VIEWS, BrowseViewId } from "./browse-views";

type ViewSwitcherProps = {
  view: BrowseViewId;
  // Left out while searching: the summary does not count the results.
  counts?: Record<BrowseViewId, number>;
  emptyViews?: ReadonlySet<BrowseViewId>;
  onChange: (view: BrowseViewId) => void;
  className?: string;
};

/**
 * Tabs to browse a directory by kind of content, each with the number of files
 * of that kind it holds. Kinds the directory surely has none of are disabled.
 */
export function ViewSwitcher({
  view,
  counts,
  emptyViews,
  onChange,
  className,
}: ViewSwitcherProps) {
  const items = BROWSE_VIEWS.map((item) => {
    const count = counts?.[item.id];
    return {
      id: item.id,
      title: item.name,
      icon: item.icon,
      tag: count,
      isDisabled: emptyViews?.has(item.id) ?? false,
    };
  });

  return (
    <Tab
      variant="underlined"
      ariaLabel="Browse by kind of content"
      className={className}
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
