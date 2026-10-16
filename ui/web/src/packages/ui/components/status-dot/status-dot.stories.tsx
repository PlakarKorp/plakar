import type { Meta, StoryObj } from "@storybook/react-vite";

import { StatusDot } from "./status-dot";
import { StatusDotVariant } from "./status-dot.constants";

const meta = {
  title: "design-system/StatusDot",
  tags: ["autodocs"],
  component: StatusDot,
  argTypes: {
    variant: {
      control: { type: "radio" },
      options: StatusDotVariant,
    },
  },
} satisfies Meta<typeof StatusDot>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    variant: "success",
    pending: false,
  },
};
