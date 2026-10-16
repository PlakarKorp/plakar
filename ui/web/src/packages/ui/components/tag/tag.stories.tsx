import type { Meta, StoryObj } from "@storybook/react-vite";

import { IconTypes } from "..";
import { Tag, TagOverflowList as TagOverflow } from "./tag";
import { TagRadius, TagSize, TagVariant } from "./tag.constants";

const meta = {
  title: "design-system/Tag",
  tags: ["autodocs"],
  component: Tag,
  argTypes: {
    variant: {
      control: { type: "select" },
      options: [undefined, ...TagVariant],
    },
    size: {
      control: { type: "radio" },
      options: TagSize,
    },
    leftIcon: {
      control: { type: "select" },
      options: [undefined, ...IconTypes],
    },
    rightIcon: {
      control: { type: "select" },
      options: [undefined, ...IconTypes],
    },
    radius: {
      control: { type: "radio" },
      options: TagRadius,
    },
  },
} satisfies Meta<typeof Tag>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    variant: "default",
    children: "Label",
    leftIcon: "bars",
    rightIcon: "bars",
  },
};

export const AllVariants: Story = {
  render: (args) => {
    return (
      <div className="flex flex-col gap-4">
        <div className="flex flex-row flex-wrap gap-4">
          {TagVariant.map((variant) => (
            <Tag
              key={`rounded-${variant}`}
              variant={variant}
              size={args.size}
              radius="rounded"
              leftIcon={args.leftIcon}
              rightIcon={args.rightIcon}
            >
              {args.children}
            </Tag>
          ))}
        </div>

        <div className="flex flex-row flex-wrap gap-4">
          {TagVariant.map((variant) => (
            <Tag
              key={`sqrt-${variant}`}
              variant={variant}
              size={args.size}
              radius="sqrt"
              leftIcon={args.leftIcon}
              rightIcon={args.rightIcon}
            >
              {args.children}
            </Tag>
          ))}
        </div>
      </div>
    );
  },
  args: {
    children: "Label",
    leftIcon: "bars",
    rightIcon: "bars",
    size: "md",
  },
  parameters: {
    controls: { hideNoControlsWarning: true },
  },
};

export const Interactive: Story = {
  name: "Interactive (link)",
  args: {
    children: "Acme",
    variant: "default",
    radius: "sqrt",
    size: "md",
  },
  render: (args) => (
    <div className="flex flex-row flex-wrap items-center gap-4">
      <Tag {...args} />
      <Tag {...args} href="#organization" />
      <Tag {...args} onPress={() => alert("pressed")}>
        {args.children} (onPress)
      </Tag>
    </div>
  ),
  parameters: {
    controls: { exclude: ["href", "onPress"] },
  },
};

export const TagOverflowList: Story = {
  args: {
    variant: "blue",
    size: "sm",
    radius: "sqrt",
  },
  parameters: {
    controls: { exclude: ["leftIcon", "rightIcon"] },
  },
  render: (args) => {
    return (
      <TagOverflow
        items={[
          "Production",
          "Critical",
          "Database",
          "Backend",
          "Frontend",
          "LongTagExampleThatMightOverflow",
          "AnotherTag",
          "Tag",
          "Example",
          "Test",
          "Sample",
          "Demo",
          "Staging",
          "Development",
          "QA",
          "UAT",
          "Hotfix",
          "Release",
          "Feature",
          "Bugfix",
        ]}
        {...args}
      />
    );
  },
};

export const TagOverflowListWithIcons: Story = {
  args: {
    variant: "blue",
    size: "md",
    radius: "sqrt",
  },
  parameters: {
    controls: { exclude: ["leftIcon", "rightIcon"] },
  },
  render: (args) => {
    return (
      <TagOverflow
        items={[
          { label: "Production", leftIcon: "aws" },
          { label: "Staging", leftIcon: "gcp" },
          { label: "Edge cluster", leftIcon: "k8s" },
          { label: "Legacy datacenter", leftIcon: "vmware" },
          { label: "Untagged" },
        ]}
        {...args}
      />
    );
  },
};
