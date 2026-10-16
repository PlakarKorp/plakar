import type { Meta, StoryObj } from "@storybook/react-vite";

import { TextArea } from "./text-area";

const meta = {
  title: "design-system/Text Area",
  tags: ["autodocs"],
  args: {
    label: "Label",
    placeholder: "Placeholder text",
    supportText: "Support text",
    className: "max-w-80",
    isDisabled: false,
    isInvalid: false,
    isRequired: false,
  },
  component: TextArea,
} satisfies Meta<typeof TextArea>;

export default meta;

type Story = StoryObj<typeof meta>;

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

export const ReadOnly: Story = {
  args: {
    value: "Read-only value",
    isReadOnly: true,
  },
};
