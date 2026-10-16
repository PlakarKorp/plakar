import { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { RemovableTags } from "./removable-tags";

type TemplateProps = {
  value: string;
};

function Template({ value }: TemplateProps) {
  const [filters, setFilters] = useState(
    Array.from({ length: 50 }, (_, idx) => idx)
  );

  return (
    <RemovableTags
      label="Filters"
      onRemove={(keys) => {
        setFilters((prev) => prev.filter((filter) => !keys.has(filter)));
      }}
    >
      {filters.map((filter) => (
        <RemovableTags.Item
          id={filter}
          key={filter.toString()}
          value={`${value} ${filter}`}
        />
      ))}
    </RemovableTags>
  );
}

const meta = {
  title: "design-system/Removable Tags",
  tags: ["autodocs"],
  component: Template,
  args: {
    value: "Value",
  },
} satisfies Meta<TemplateProps>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
