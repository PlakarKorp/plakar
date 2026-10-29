import { Meta, StoryObj } from "@storybook/react-vite";

import { ActivityDay, ActivityHeatmap } from "./activity-heatmap";

function lastDays(counts: number[]): ActivityDay[] {
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  return counts.map((count, index) => {
    const date = new Date(today);
    date.setDate(today.getDate() - (counts.length - 1 - index));
    return { date, count };
  });
}

const meta = {
  title: "design-system/Activity Heatmap",
  tags: ["autodocs"],
  component: ActivityHeatmap,
  args: {
    days: lastDays([
      0, 0, 1, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 3, 1, 0, 0, 0,
      0, 0, 0, 1, 0,
    ]),
  },
} satisfies Meta<typeof ActivityHeatmap>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Empty: Story = {
  args: {
    days: lastDays(Array.from({ length: 30 }, () => 0)),
  },
};

export const HighVolume: Story = {
  args: {
    days: lastDays(
      Array.from({ length: 30 }, (_, index) => ((index * 7) % 11) * 4)
    ),
  },
};

export const Widths: Story = {
  render: (args) => (
    <div className="flex flex-col gap-6">
      {[240, 320, 480, 720].map((width) => (
        <div key={width} style={{ width }}>
          <ActivityHeatmap {...args} />
        </div>
      ))}
    </div>
  ),
};
