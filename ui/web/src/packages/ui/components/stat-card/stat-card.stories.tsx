import { Meta, StoryObj } from "@storybook/react-vite";

import { StatCard } from "./stat-card";
import { Typography } from "../typography/typography";

const meta = {
  title: "design-system/Stat Card",
  tags: ["autodocs"],
  component: StatCard,
  args: {
    title: "Restore points",
    value: "74",
    label: "total",
  },
} satisfies Meta<typeof StatCard>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const WithFooter: Story = {
  args: {
    footer: (
      <Typography size="xs" weight="semibold" className="text-success">
        +7 over the last 30 days
      </Typography>
    ),
  },
};

export const Grid: Story = {
  render: () => (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <StatCard title="Restore points" value="74" label="total" />
      <StatCard title="Logical size" value="90.8 GB" label="of data" />
      <StatCard title="Storage size" value="6.51 GB" label="on disk" />
      <StatCard title="Efficiency" value="93%" label="space saved" />
    </div>
  ),
};
