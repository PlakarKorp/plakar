import { IconTypes } from "@plakar-ui/ui/components";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { IconBadge } from "./icon-badge";
import { IconBadgeTypes } from "./icon-badge.constants";

const meta = {
  title: "Components/Icon Badge",
  tags: ["autodocs"],
  args: {
    icon: "database",
    type: "default",
    loading: false,
  },
  argTypes: {
    icon: {
      control: { type: "select" },
      options: IconTypes,
      table: {
        type: { summary: "IconType" },
      },
    },
    type: {
      control: "select",
      options: IconBadgeTypes,
    },
    size: {
      control: "select",
      options: ["sm", "md"],
    },
  },
  component: IconBadge,
} satisfies Meta<typeof IconBadge>;

export default meta;

type Story = StoryObj<typeof meta>;

export const PlakarIcons: Story = {};

export const Skeleton: Story = {
  args: {
    isSkeleton: true,
  },
};

export const AllColors: Story = {
  parameters: {
    controls: { disable: true },
  },
  render: () => (
    <div className="flex flex-wrap gap-4">
      <IconBadge icon="house" type="default" />
      <IconBadge icon="arrows-rotate" type="accent" />
      <IconBadge icon="user" type="brand" />
      <IconBadge icon="robot" type="service" />
      <IconBadge icon="upload" type="source" />
      <IconBadge icon="download" type="destination" />
      <IconBadge icon="box-archive" type="store" />
      <IconBadge icon="lock" type="secret_provider" />
      <IconBadge icon="lock" type="disabled" />
      <IconBadge icon="lock" type="error" />
      <IconBadge icon="lock" type="warning" />
      <IconBadge icon="plug-circle-check" type="default" />
      <IconBadge icon="plug-circle-xmark" type="default" />
    </div>
  ),
};
