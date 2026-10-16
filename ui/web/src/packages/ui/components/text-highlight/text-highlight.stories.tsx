import type { Meta, StoryObj } from "@storybook/react-vite";

import { Typography } from "..";
import { TextHighlight } from "./text-highlight";

const meta = {
  title: "design-system/Text Highlight",
  component: TextHighlight,
  tags: ["autodocs"],
  argTypes: {
    text: {
      control: { type: "text" },
      description: "The text content to search within",
    },
    search: {
      control: { type: "text" },
      description: "The text to search and highlight within the text",
    },
  },
} satisfies Meta<typeof TextHighlight>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    text: "This is some example text to highlight",
    search: "example",
  },
  render: (args) => (
    <Typography className="text-primary">
      <TextHighlight {...args} />
    </Typography>
  ),
};
