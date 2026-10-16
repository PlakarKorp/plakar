import type { Meta, StoryObj } from "@storybook/react-vite";

import { NumberInput } from "./number-input";
import { Button } from "../button/button";

const meta = {
  title: "design-system/NumberInput",
  tags: ["autodocs"],
  args: {
    isDisabled: false,
    isInvalid: false,
    isRequired: false,
  },
  component: NumberInput,
} satisfies Meta<typeof NumberInput>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    label: "Label",
    placeholder: "Placeholder text",
    supportText: "Support text",
    defaultValue: 0,
    minValue: 0,
    maxValue: 1024,
    className: "w-40",
  },
};

export const Disabled: Story = {
  args: {
    ...Default.args,
    isDisabled: true,
  },
};

export const ReadOnly: Story = {
  args: {
    ...Default.args,
    isReadOnly: true,
  },
};

export const Invalid: Story = {
  args: {
    ...Default.args,
    isInvalid: true,
  },
};

export const WithLabelAction: Story = {
  args: {
    ...Default.args,
    labelAction: (
      <Button variant="tertiary" size="sm" leftIcon="arrow-rotate-left">
        Reset
      </Button>
    ),
  },
};
