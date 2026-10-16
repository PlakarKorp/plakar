import type { Meta, StoryObj } from "@storybook/react-vite";

import { LoginBanner } from "./login-banner";

const meta = {
  title: "design-system/Login Banner",
  tags: ["autodocs"],
  component: LoginBanner,
  argTypes: {},
} satisfies Meta<typeof LoginBanner>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    href: "#",
  },
};

export const WithButton: Story = {
  args: {
    onClick: () => alert("clicked"),
  },
};
