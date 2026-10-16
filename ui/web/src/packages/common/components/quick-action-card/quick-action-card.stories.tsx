import { Button, IconTypes } from "@plakar-ui/ui/components";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { QuickActionCard } from "./quick-action-card";

const meta = {
  title: "design-system/Quick Action Card",
  tags: ["autodocs"],
  component: QuickActionCard,
  argTypes: {
    icon: {
      control: { type: "select" },
      options: IconTypes,
    },
    as: {
      control: false,
      table: { disable: true },
    },
    action: {
      control: false,
      table: { disable: true },
    },
  },
} satisfies Meta<typeof QuickActionCard>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    title: "Create Backup",
    description: "Back up your data to prevent loss.",
    icon: "upload",
  },
};

export const Disabled: Story = {
  args: {
    title: "Create Backup",
    description: "Back up your data to prevent loss.",
    icon: "upload",
    isDisabled: true,
  },
};

export const WithAction: Story = {
  args: {
    interactive: false,
    title: "Amazon S3",
    description: "Object storage integration.",
    icon: "aws",
    iconBadgeType: "default",
    action: (
      <Button size="sm" leftIcon="download">
        Install
      </Button>
    ),
  },
};
