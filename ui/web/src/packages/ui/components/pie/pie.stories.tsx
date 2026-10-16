import { Meta, StoryObj } from "@storybook/react-vite";

import { Pie } from "./pie";

const meta = {
  title: "design-system/Pie",
  tags: ["autodocs"],
  component: Pie,
  argTypes: {
    value: {
      control: { type: "number" },
    },
    size: {
      control: { type: "number" },
    },
    strokeWidth: {
      control: { type: "number" },
    },
  },
} satisfies Meta<typeof Pie>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    value: 33,
    size: 84,
    strokeWidth: 8,
  },
};
