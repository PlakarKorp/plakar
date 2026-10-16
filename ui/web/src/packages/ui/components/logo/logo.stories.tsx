import { Meta, StoryObj } from "@storybook/react-vite";

import { Logo } from "./logo";
import { LogoVariants } from "./types";

const meta = {
  title: "design-system/Logo",
  tags: ["autodocs"],
  component: Logo,
  args: {
    className: "h-16",
  },
  argTypes: {
    variant: {
      control: { type: "select" },
      options: [undefined, ...LogoVariants],
    },
    className: { control: "text" },
  },
  render: (args) => <Logo {...args} />,
} satisfies Meta<typeof Logo>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Icon: Story = {
  args: {
    variant: "icon",
  },
};

export const Full: Story = {
  args: {
    variant: "full",
  },
};

export const ControlPlaneIcon: Story = {
  args: {
    variant: "icon-control-plane",
  },
};

export const ControlPlaneFull: Story = {
  args: {
    variant: "full-control-plane",
  },
};
