import { Meta, StoryObj } from "@storybook/react-vite";

import { AreaChart } from "./area-chart";

const meta = {
  title: "design-system/AreaChart",
  tags: ["autodocs"],
  component: AreaChart,
  argTypes: {
    data: {
      control: { type: "object" },
    },
    width: {
      control: { type: "number" },
    },
    height: {
      control: { type: "number" },
    },
    strokeWidth: {
      control: { type: "number" },
    },
  },
} satisfies Meta<typeof AreaChart>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    data: [
      { value: 10 },
      { value: 40 },
      { value: 30 },
      { value: 70 },
      { value: 50 },
      { value: 20 },
      { value: 45 },
    ],
    width: 120,
    height: 84,
    strokeWidth: 2,
    padding: 0,
  },
};

export const Empty: Story = {
  args: {
    data: [
      { value: 0 },
      { value: 0 },
      { value: 0 },
      { value: 0 },
      { value: 0 },
      { value: 0 },
      { value: 0 },
    ],
    width: 120,
    height: 84,
    strokeWidth: 2,
    padding: 0,
  },
};
