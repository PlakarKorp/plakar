import type { Meta, StoryObj } from "@storybook/react-vite";

import { CountBadge } from "./count-badge";

const meta = {
  title: "design-system/Count Badge",
  tags: ["autodocs"],
  component: CountBadge,
  args: {
    children: 115,
    isOnAccent: false,
  },
  decorators: [
    (Story, { args }) => (
      <div
        className={
          args.isOnAccent
            ? "bg-accent inline-flex rounded-lg p-2"
            : "inline-flex p-2"
        }
      >
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof CountBadge>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const OnAccent: Story = {
  args: {
    isOnAccent: true,
  },
};
