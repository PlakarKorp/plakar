import type { Meta, StoryObj } from "@storybook/react-vite";

import { TagsInput } from "./tags-input";

const meta: Meta<typeof TagsInput> = {
  title: "design-system/Tags Input",
  component: TagsInput,
  tags: ["autodocs"],
  args: {
    label: "Tags",
    placeholder: "Add a tag…",
    tags: ["production", "design", "development"],
    supportText: "Support text",
    isDisabled: false,
    isInvalid: false,
    isRequired: false,
  },
};
export default meta;

type Story = StoryObj<typeof TagsInput>;

export const Default: Story = {};

export const Disabled: Story = {
  args: {
    isDisabled: true,
  },
};

export const Invalid: Story = {
  args: {
    isInvalid: true,
  },
};

export const IsReadOnly: Story = {
  args: {
    isReadOnly: true,
  },
};

export const WithDuplicates: Story = {
  args: {
    tags: ["same tag", "same tag"],
    supportText: "Duplicate tags are allowed if provided initially",
  },
};
