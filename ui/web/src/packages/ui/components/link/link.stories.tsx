import type { Meta, StoryObj } from "@storybook/react-vite";

import { Link } from "./link";

const meta = {
  title: "design-system/Link",
  tags: ["autodocs"],
  args: {
    children: "This is a link",
    href: "https://plakar.io",
  },
  component: Link,
} satisfies Meta<typeof Link>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const OnPress: Story = {
  render: ({ children }) => (
    <Link onPress={() => alert("Link pressed")}>{children}</Link>
  ),
};

export const InheritColor: Story = {
  args: {
    inheritColor: true,
  },
  render: (args) => (
    <div className="text-red-500">
      <Link {...args} />
    </div>
  ),
};
