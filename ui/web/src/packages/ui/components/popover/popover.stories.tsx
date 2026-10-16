import { Meta, StoryObj } from "@storybook/react-vite";

import { Popover } from "./popover";

const meta = {
  title: "design-system/Popover",
  tags: ["autodocs"],
  component: Popover,
  argTypes: {
    children: {
      control: false,
      table: { disable: true }, // Hide children from the args table
    },
  },
} satisfies Meta<typeof Popover>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    label: "This is a popover.",
    children: <div>Click me</div>,
    maxWidth: "200px",
  },
};
