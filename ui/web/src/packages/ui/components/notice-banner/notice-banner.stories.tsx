import { Meta, StoryObj } from "@storybook/react-vite";

import { NoticeBanner } from "./notice-banner";

const meta = {
  title: "design-system/NoticeBanner",
  tags: ["autodocs"],
  component: NoticeBanner,
  argTypes: {
    children: {
      control: false,
      table: { disable: true },
    },
    onDismiss: {
      action: "dismissed",
      table: { disable: true },
    },
    onOpenChange: {
      action: "open changed",
      table: { disable: true },
    },
  },
} satisfies Meta<typeof NoticeBanner>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    variant: "info",
    dismissible: true,
    defaultOpen: true,
    children: <>We’ve updated our pricing.</>,
  },
};

export const Danger: Story = {
  args: {
    variant: "danger",
    dismissible: true,
    defaultOpen: true,
    children: <>We’re currently experiencing a service outage.</>,
  },
};

export const NotDismissible: Story = {
  args: {
    variant: "info",
    dismissible: false,
    defaultOpen: true,
    children: <>This banner cannot be dismissed.</>,
  },
};
