import { Meta, StoryObj } from "@storybook/react-vite";

import { Button } from "..";
import { Tooltip } from "./tooltip";

const meta = {
  title: "design-system/Tooltip",
  tags: ["autodocs"],
  component: Tooltip,
  argTypes: {
    children: {
      control: false,
      table: { disable: true }, // Hide children from the args table
    },
  },
} satisfies Meta<typeof Tooltip>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    label: "This is a tooltip.",
    children: <Button>Hover me</Button>,
    maxWidth: "200px",
  },
};
