import type { Meta, StoryObj } from "@storybook/react-vite";
import { ComponentProps } from "react";

import { NotificationList } from "./notification-list";
import { NotificationSevrities as NotificationSeverities } from "./notification-list.constants";

type TemplateProps = ComponentProps<typeof NotificationList.Item>;

function Template({ ...props }: TemplateProps) {
  return (
    <NotificationList ariaLabel="Notifications">
      {Array.from({ length: 5 }).map((_, idx) => (
        <NotificationList.Item key={idx} {...props} />
      ))}
    </NotificationList>
  );
}

const meta = {
  title: "design-system/NotificationList",
  tags: ["autodocs"],
  component: Template,
  argTypes: {
    severity: {
      control: { type: "radio" },
      options: NotificationSeverities,
    },
    date: {
      control: { type: "text" },
      description: "Relative time string, e.g. '2 min ago'",
    },
  },
} satisfies Meta<TemplateProps>;

export default meta;

type Story = StoryObj<TemplateProps>;

export const Default: Story = {
  args: {
    severity: "default",
    title: "Backup",
    description: "This is a great notification to show.",
    date: "2 min ago",
    unread: true,
  },
};
