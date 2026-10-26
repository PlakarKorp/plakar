import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { Tab, TabItem } from "./tab";
import { DropdownListItem } from "../dropdown-list/dropdown-list";
import { UpgradeIndicator } from "../upgrade-indicator/upgrade-indicator";

const meta = {
  title: "design-system/Tab",
  tags: ["autodocs"],
  component: Tab,
  argTypes: {
    selected: {
      control: false,
      table: {
        disable: true,
      },
    },
    variant: {
      options: ["underlined", "pill"],
      control: { type: "radio" },
      description: "The visual style of the tab.",
    },
  },
} satisfies Meta<typeof Tab>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Underlined: Story = {
  args: {
    ariaLabel: "List of items",
    variant: "underlined",
    items: [
      {
        id: "label-1",
        title: "Label 1",
        icon: "bars",
        tag: "12",
      },
      {
        id: "label-2",
        title: "Label 2",
        icon: "bars",
        iconNotification: <UpgradeIndicator label="Update available" />,
        tag: "5",
      },
      {
        id: "label-3",
        title: "Label 3",
        icon: "bars",
        tag: "8",
      },
      {
        id: "label-4",
        title: "Label 4",
        icon: "bars",
        tag: "3",
      },
    ],
  },
};

export const Pill: Story = {
  args: {
    ariaLabel: "List of items",
    variant: "pill",
    items: [
      {
        id: "label-1",
        title: "Label 1",
      },
      {
        id: "label-2",
        title: "Label 2",
      },
      {
        id: "label-3",
        title: "Label 3",
      },
      {
        id: "label-4",
        title: "Label 4",
      },
    ],
  },
};

export const DisabledTab: Story = {
  args: {
    ariaLabel: "List of items",
    variant: "underlined",
    items: [
      {
        id: "label-1",
        title: "Label 1",
      },
      {
        id: "label-2",
        title: "Label 2",
        isDisabled: true,
        tooltip: "Not available yet",
      },
      {
        id: "label-3",
        title: "Label 3",
      },
      {
        id: "label-4",
        title: "Label 4",
      },
    ],
  },
};

// Laid out in a narrow flex row, as on mobile: the tabs must scroll inside it.
export const Overflowing: Story = {
  args: {
    ariaLabel: "List of items",
    variant: "underlined",
    items: Array.from({ length: 10 }, (_, index) => ({
      id: `label-${index + 1}`,
      title: `Label ${index + 1}`,
      icon: "bars",
      tag: String(index * 4),
    })),
  },
  decorators: [
    (Story) => (
      <div className="flex w-80 outline outline-dashed">
        <Story />
      </div>
    ),
  ],
};

const FIXED_TABS: TabItem[] = [
  { id: "manual", title: "Manual scheduler" },
  { id: "policy", title: "Policy scheduler" },
];

const OPERATORS: DropdownListItem[] = Array.from(
  { length: 12 },
  (_, index) => ({
    id: `operator-${index}`,
    name: `k8s-operator-${index.toString().padStart(2, "0")}`,
    icon: "robot",
    description: `seen ${index + 1} min ago`,
    tag: index * 3,
  })
);

const DropdownTab = ({
  variant,
  items,
}: {
  variant: "underlined" | "pill";
  items: DropdownListItem[];
}) => {
  const [selected, setSelected] = useState<string | null>("manual");
  const [operatorId, setOperatorId] = useState<string | null>(null);

  return (
    <div className="flex items-center gap-2">
      <Tab
        variant={variant}
        ariaLabel="Schedulers"
        items={FIXED_TABS}
        selected={selected}
        onSelectionChange={(key) => setSelected(String(key))}
      />

      <Tab.Dropdown
        variant={variant}
        label="Operators"
        icon="users"
        items={items}
        tag={items.length}
        selectedId={operatorId}
        isSelected={selected === null}
        onSelect={(id) => {
          setOperatorId(id);
          setSelected(null);
        }}
        searchPlaceholder="Search operators"
        emptyTitle="No operator yet"
        emptyDescription="Operators appear here once they push their first task."
      />
    </div>
  );
};

export const WithDropdown: Story = {
  args: { ariaLabel: "Schedulers", variant: "underlined", items: FIXED_TABS },
  render: ({ variant }) => <DropdownTab variant={variant} items={OPERATORS} />,
};

export const WithEmptyDropdown: Story = {
  args: { ariaLabel: "Schedulers", variant: "underlined", items: FIXED_TABS },
  render: ({ variant }) => <DropdownTab variant={variant} items={[]} />,
};
