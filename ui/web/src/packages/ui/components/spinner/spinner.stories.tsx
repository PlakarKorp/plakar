import type { Meta, StoryObj } from "@storybook/react-vite";

import { IconSizes } from "..";
import { Spinner } from "./spinner";

const meta = {
  title: "design-system/Spinner",
  tags: ["autodocs"],
  component: Spinner,
  argTypes: {
    size: {
      control: { type: "select" },
      options: IconSizes,
    },
  },
} satisfies Meta<typeof Spinner>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    size: "6xl",
    inheritColor: false,
  },
};
