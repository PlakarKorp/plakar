import { Meta, StoryObj } from "@storybook/react-vite";

import { IllustrationTypes } from "..";
import { EmptyList } from "./empty-list";

const meta = {
  title: "design-system/Empty List",
  tags: ["autodocs"],
  component: EmptyList,
  argTypes: {
    illustrationType: {
      control: { type: "select" },
      options: [undefined, ...IllustrationTypes],
      table: {
        type: { summary: "IllustrationType" },
      },
    },
  },
} satisfies Meta<typeof EmptyList>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    title: "No items found",
    description: "Try adjusting your search or filter to find what you need.",
    illustrationType: "default",
    actions: [
      {
        children: "Retry",
        variant: "secondary",
        onPress: () => alert("Retry clicked"),
      },
      {
        children: "Other action",
        onPress: () => alert("Other action clicked"),
      },
    ],
  },
};

export const WithActions: Story = {
  args: {
    ...Default.args,
    illustrationType: undefined,
  },
};

export const WithIllustration: Story = {
  args: {
    ...Default.args,
    actions: undefined,
  },
};

export const Simple: Story = {
  args: {
    ...Default.args,
    illustrationType: undefined,
    actions: undefined,
  },
};

export const Small: Story = {
  args: {
    ...Default.args,
    illustrationType: undefined,
    size: "sm",
  },
};
