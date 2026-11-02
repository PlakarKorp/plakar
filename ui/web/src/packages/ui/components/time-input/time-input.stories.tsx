import { Time } from "@internationalized/date";
import { Meta, StoryObj } from "@storybook/react-vite";

import { TimeInput } from "./time-input";

const meta = {
  title: "design-system/Time Input",
  tags: ["autodocs"],
  component: TimeInput,
  args: {
    label: "Start time",
    supportText: "",
    inputWidth: "w-28",
    isDisabled: false,
    isReadOnly: false,
    isInvalid: false,
    isRequired: false,
  },
} satisfies Meta<typeof TimeInput>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const WithValue: Story = {
  args: {
    defaultValue: new Time(2, 30),
  },
};

export const Disabled: Story = {
  args: {
    isDisabled: true,
  },
};

export const Invalid: Story = {
  args: {
    isInvalid: true,
    supportText: "This field is required.",
  },
};

export const ReadOnly: Story = {
  args: {
    isReadOnly: true,
    value: new Time(2, 30),
  },
};
