import { Meta, StoryObj } from "@storybook/react-vite";

import { Checkbox } from "./checkbox";
import { CheckboxSizes } from "./checkbox.constants";

const meta = {
  title: "design-system/Checkbox",
  tags: ["autodocs"],
  component: Checkbox,
  argTypes: {
    size: {
      control: { type: "select" },
      options: [undefined, ...CheckboxSizes],
      table: {
        type: { summary: CheckboxSizes.join(" | ") },
      },
    },
  },
} satisfies Meta<typeof Checkbox>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    label: "Label",
    description: "Description",
    isDisabled: false,
    isSelected: false,
    isIndeterminate: false,
  },
};

export const Disabled: Story = {
  args: {
    ...Default.args,
    isDisabled: true,
  },
};

export const Checked: Story = {
  args: {
    ...Default.args,
    isSelected: true,
  },
};

export const Small: Story = {
  args: {
    ...Default.args,
    size: "sm",
  },
};

export const WithoutLabel: Story = {
  args: {},
};

export const Indeterminate: Story = {
  args: {
    ...Default.args,
    isIndeterminate: true,
  },
};
