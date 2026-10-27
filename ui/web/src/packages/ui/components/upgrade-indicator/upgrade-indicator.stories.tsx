import type { Meta, StoryObj } from "@storybook/react-vite";

import { UpgradeIndicator } from "./upgrade-indicator";

const meta = {
  title: "design-system/Upgrade Indicator",
  tags: ["autodocs"],
  component: UpgradeIndicator,
  args: {
    label: "3 integrations to upgrade",
  },
} satisfies Meta<typeof UpgradeIndicator>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
