import { Meta, StoryObj } from "@storybook/react-vite";

import { Illustration, Typography } from "..";
import { Hint } from "./hint";

const meta: Meta<typeof Hint> = {
  title: "design-system/Hint",
  component: Hint,
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof Hint>;

export const Default: Story = {
  args: {
    tooltip: "My hint text",
    children: "My text with a hint",
    color: "accent",
    underline: "always",
  },
};

export const InsideText: Story = {
  args: {
    tooltip: "My hint text",
    children: "My text with a hint",
    color: "accent",
    underline: "always",
  },
  render: (args) => (
    <Typography as="p" className="text-primary">
      This is a paragraph with a hint inside. <Hint {...args} /> This is the end
      of the paragraph.
    </Typography>
  ),
};

export const CustomTooltipContent: Story = {
  args: {
    tooltip: <Illustration type="waving" />,
    children: "Custom tooltip content",
    color: "accent",
    underline: "always",
  },
};
