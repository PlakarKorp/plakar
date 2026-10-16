import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { ButtonDropdown } from "./button-dropdown";
import { DropdownList, DropdownListItem } from "../dropdown-list/dropdown-list";

const INVENTORIES: DropdownListItem[] = Array.from(
  { length: 12 },
  (_, index) => ({
    id: `inventory-${index}`,
    name:
      index === 3
        ? "Inventory with a very long name that needs to be truncated"
        : `Inventory ${index.toString().padStart(2, "0")}`,
    icon: "cubes",
    description: `${index * 7} resources · Synced ${index + 1} minutes ago`,
    action:
      index % 3 === 0
        ? { label: "Add resource to", icon: "plus" }
        : {
            label: index === 1 ? "Syncing" : "Sync",
            icon: "arrows-rotate",
            isLoading: index === 1,
          },
    isDisabled: index === 1,
  })
);

type TemplateProps = {
  items: DropdownListItem[];
  onSelect: (id: string) => void;
  isLoading?: boolean;
  isDisabled?: boolean;
};

function Template({ items, onSelect, isLoading, isDisabled }: TemplateProps) {
  return (
    <ButtonDropdown
      label="Update"
      leftIcon="arrows-rotate"
      isDisabled={isDisabled}
    >
      {(close) => (
        <DropdownList
          label="Inventories"
          items={items}
          onSelect={(id) => {
            onSelect(id);
            close();
          }}
          searchPlaceholder="Search inventories"
          emptyTitle="No inventory yet"
          emptyDescription="Add an inventory to sync or register resources."
          isLoading={isLoading}
        />
      )}
    </ButtonDropdown>
  );
}

const meta = {
  title: "design-system/ButtonDropdown",
  tags: ["autodocs"],
  component: Template,
  args: {
    items: INVENTORIES,
    onSelect: fn(),
  },
} satisfies Meta<typeof Template>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Empty: Story = {
  args: { items: [] },
};

export const Loading: Story = {
  args: { items: [], isLoading: true },
};

export const Disabled: Story = {
  args: { isDisabled: true },
};
