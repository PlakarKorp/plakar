import { SegmentedControl, Select } from "@plakar-ui/ui/components";
import { ComponentProps } from "react";

import { FilterPanel } from "../filter-panel/filter-panel";

const ALL = "all";

type SegmentedControlItem = ComponentProps<
  typeof SegmentedControl
>["items"][number];

type IntegrationsFiltersProps<Type extends string, Status extends string> = {
  search: string;
  onSearchChange: (search: string) => void;
  status: Status | undefined;
  statusItems: Array<SegmentedControlItem & { id: Status }>;
  allCount: number | undefined;
  onStatusChange: (status: Status | undefined) => void;
  types: Type[] | undefined;
  typeItems: Array<{ id: Type; label: string }>;
  onTypesChange: (types: Type[] | undefined) => void;
  shown: number | undefined;
  total: number | undefined;
  onClear: (() => void) | undefined;
};

export function IntegrationsFilters<
  Type extends string,
  Status extends string,
>({
  search,
  onSearchChange,
  status,
  statusItems,
  allCount,
  onStatusChange,
  types,
  typeItems,
  onTypesChange,
  shown,
  total,
  onClear,
}: IntegrationsFiltersProps<Type, Status>) {
  return (
    <FilterPanel>
      <FilterPanel.Search
        value={search}
        onSearch={onSearchChange}
        placeholder="Search integrations by name…"
        aria-label="Search integrations"
      />
      <FilterPanel.Groups>
        <FilterPanel.Group label="Status">
          <SegmentedControl
            variant="accent"
            ariaLabel="Status"
            selectedId={status ?? ALL}
            onSelect={(id) =>
              onStatusChange(statusItems.find((item) => item.id === id)?.id)
            }
            items={[{ id: ALL, label: "All", count: allCount }, ...statusItems]}
          />
        </FilterPanel.Group>
        <FilterPanel.Group label="Type">
          <Select
            aria-label="Type"
            value={types?.length === 1 ? types[0] : ALL}
            onChange={(key) => {
              const type = typeItems.find((item) => item.id === key)?.id;
              onTypesChange(type === undefined ? undefined : [type]);
            }}
            className="min-w-60"
          >
            <Select.ListBox>
              {[{ id: ALL, label: "All types" }, ...typeItems].map(
                ({ id, label }) => (
                  <Select.Item key={id} id={id} name={label} />
                )
              )}
            </Select.ListBox>
          </Select>
        </FilterPanel.Group>
      </FilterPanel.Groups>
      <FilterPanel.Footer
        shown={shown}
        total={total}
        noun="integrations"
        onClear={onClear}
      />
    </FilterPanel>
  );
}
