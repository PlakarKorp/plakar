import {
  CountBadge,
  Icon,
  SegmentedControl,
  Select,
  Typography,
} from "@plakar-ui/ui/components";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { FilterPanel } from "./filter-panel";

const PROTECTION = [
  { id: "all", label: "All", count: 115 },
  { id: "protected", label: "Protected", count: 4, dot: "success" as const },
  {
    id: "unprotected",
    label: "Not protected",
    count: 108,
    dot: "warning" as const,
  },
  { id: "unmanaged", label: "Unmanaged", count: 0 },
];

const CLASSES = [
  { id: "all", name: "All resource classes", count: 115 },
  { id: "database", name: "Databases", count: 42 },
  { id: "storage", name: "Object storage", count: 73 },
];

const SORTS = [
  { id: "class", label: "Class" },
  { id: "name", label: "Name" },
];

function FilterPanelDemo({
  withFooter,
  searchOnly,
}: {
  withFooter: boolean;
  searchOnly?: boolean;
}) {
  const [search, setSearch] = useState("");
  const [protection, setProtection] = useState("all");
  const [resourceClass, setResourceClass] = useState("all");
  const [sort, setSort] = useState("class");

  const isFiltered =
    !!search || protection !== "all" || resourceClass !== "all";

  const searchField = (
    <FilterPanel.Search
      value={search}
      onSearch={setSearch}
      placeholder="Search by name or URN…"
      aria-label="Search resources"
    />
  );

  if (searchOnly) {
    return <FilterPanel>{searchField}</FilterPanel>;
  }

  return (
    <FilterPanel>
      {searchField}
      <FilterPanel.Groups>
        <FilterPanel.Group label="Protection">
          <SegmentedControl
            variant="accent"
            ariaLabel="Protection"
            items={PROTECTION}
            selectedId={protection}
            onSelect={setProtection}
          />
        </FilterPanel.Group>
        <FilterPanel.Group label="Resource class">
          <Select
            aria-label="Resource class"
            value={resourceClass}
            onChange={(key) => setResourceClass(String(key))}
            className="min-w-60"
          >
            <Select.ListBox>
              {CLASSES.map(({ id, name, count }) => (
                <Select.Item
                  key={id}
                  id={id}
                  name={name}
                  item={
                    <span className="text-primary flex items-center gap-2">
                      <Icon type="layer-group" size="md" />
                      <Typography as="span" size="sm" className="flex-1">
                        {name}
                      </Typography>
                      <CountBadge>{count}</CountBadge>
                    </span>
                  }
                />
              ))}
            </Select.ListBox>
          </Select>
        </FilterPanel.Group>
      </FilterPanel.Groups>
      {withFooter && (
        <FilterPanel.Footer
          shown={115}
          total={115}
          noun="resources"
          onClear={
            isFiltered
              ? () => {
                  setSearch("");
                  setProtection("all");
                  setResourceClass("all");
                }
              : undefined
          }
          sort={{ items: SORTS, selectedId: sort, onSelect: setSort }}
        />
      )}
    </FilterPanel>
  );
}

const meta = {
  title: "Components/Filter Panel",
  tags: ["autodocs"],
  component: FilterPanelDemo,
  args: {
    withFooter: true,
  },
} satisfies Meta<typeof FilterPanelDemo>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const WithoutFooter: Story = {
  args: {
    withFooter: false,
  },
};

export const SearchOnly: Story = {
  args: {
    withFooter: false,
    searchOnly: true,
  },
};
