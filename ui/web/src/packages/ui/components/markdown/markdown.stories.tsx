import type { Meta, StoryObj } from "@storybook/react-vite";

import { Markdown } from "./markdown";
import MARKDOWN_CONTENT from "./markdown.stories.example.md?raw";

const meta = {
  title: "design-system/Markdown",
  tags: ["autodocs"],
  component: Markdown,
  argTypes: {},
} satisfies Meta<typeof Markdown>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    content: MARKDOWN_CONTENT,
  },
};
