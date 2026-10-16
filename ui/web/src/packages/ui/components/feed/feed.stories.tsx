import type { Meta, StoryObj } from "@storybook/react-vite";
import { ComponentProps } from "react";

import { IconTypes } from "..";
import { Feed } from "./feed";
import { FeedItemStatuses } from "./feed.constants";

type TemplateProps = ComponentProps<typeof Feed.Item> & {
  loading: boolean;
};

function Template({ loading, ...props }: TemplateProps) {
  return (
    <Feed loading={loading}>
      {Array.from({ length: 5 }).map((_, idx) => (
        <Feed.Item key={idx} {...props} />
      ))}
    </Feed>
  );
}

const meta = {
  title: "design-system/Feed",
  tags: ["autodocs"],
  component: Template,
  argTypes: {
    icon: {
      control: { type: "select" },
      options: [undefined, ...IconTypes],
    },
    status: {
      control: { type: "radio" },
      options: FeedItemStatuses,
    },
  },
} satisfies Meta<TemplateProps>;

export default meta;

type Story = StoryObj<TemplateProps>;

export const Default: Story = {
  args: {
    loading: false,
    icon: "cloud",
    status: "default",
    title: "Backup",
    description: "Successfully done",
    alt: "2 min ago",
  },
};
