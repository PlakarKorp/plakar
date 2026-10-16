import type { Meta, StoryObj } from "@storybook/react-vite";
import { ComponentProps } from "react";
import { fn } from "storybook/test";

import { IconTypes, LoginBanner } from "..";
import { Menu } from "./menu";

type TemplateProps = ComponentProps<typeof Menu.Item> & {
  loading: boolean;
  showHeader: boolean;
  unstyled?: boolean;
  itemsCount: number;
};

function Template({
  loading,
  showHeader,
  unstyled,
  itemsCount,
  ...props
}: TemplateProps) {
  const banner = showHeader ? <LoginBanner /> : undefined;

  return (
    <Menu loading={loading} header={banner} unstyled={unstyled}>
      {Array.from({ length: itemsCount }).map((_, idx) => (
        <Menu.Item key={idx} {...props} />
      ))}
    </Menu>
  );
}

const meta = {
  title: "design-system/Menu",
  tags: ["autodocs"],
  component: Template,
  argTypes: {
    leftIcon: {
      control: { type: "select" },
      options: [undefined, ...IconTypes],
    },
    rightIcon: {
      control: { type: "select" },
      options: [undefined, ...IconTypes],
    },
    showHeader: {
      control: { type: "boolean" },
      description: "Whether to show a header in the list or not",
    },
    unstyled: {
      control: { type: "boolean" },
    },
    itemsCount: {
      control: { type: "number" },
      description: "How many items to render in the list",
    },
  },
} satisfies Meta<TemplateProps>;

export default meta;

type Story = StoryObj<TemplateProps>;

export const Default: Story = {
  args: {
    loading: false,
    leftIcon: "bars",
    label: "Lorem ipsum",
    altText: "2 months ago",
    rightIcon: "bars",
    error: false,
    onAction: fn(),
    showHeader: true,
    itemsCount: 5,
  },
};

export const Unstyled: Story = {
  args: {
    ...Default.args,
    showHeader: false,
    unstyled: true,
  },
};

export const Scrollable: Story = {
  args: {
    ...Default.args,
    showHeader: false,
    itemsCount: 20,
  },
};

export const WithDescription: Story = {
  args: {
    ...Default.args,
    description: "Dolor sit amet, consectetur adipiscing elit",
  },
};
