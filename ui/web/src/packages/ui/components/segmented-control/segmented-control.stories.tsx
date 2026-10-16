import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { SegmentedControl } from "./segmented-control";

const themeItems = [
  { id: "light", label: "Light", icon: "sun" as const },
  { id: "system", label: "System", icon: "circle-half-stroke" as const },
  { id: "dark", label: "Dark", icon: "moon" as const },
];

const meta = {
  title: "design-system/Segmented Control",
  tags: ["autodocs"],
  args: {
    items: themeItems,
    selectedId: "system",
    onSelect: () => {},
    ariaLabel: "Theme",
    hideLabels: false,
    isDisabled: false,
  },
  component: SegmentedControl,
  render: (args) => {
    // eslint-disable-next-line react-hooks/rules-of-hooks
    const [selectedId, setSelectedId] = useState(args.selectedId);
    return (
      <SegmentedControl
        {...args}
        selectedId={selectedId}
        onSelect={setSelectedId}
      />
    );
  },
} satisfies Meta<typeof SegmentedControl>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const IconsOnly: Story = {
  args: {
    hideLabels: true,
  },
};

export const Disabled: Story = {
  args: {
    isDisabled: true,
  },
};

export const TwoSegments: Story = {
  args: {
    items: [
      { id: "list", label: "List", icon: "list" },
      { id: "tree", label: "Tree", icon: "diagram-project" },
    ],
    selectedId: "list",
    ariaLabel: "Layout",
  },
};
