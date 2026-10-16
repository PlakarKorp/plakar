import { parseDateTime } from "@internationalized/date";
import { Meta, StoryObj } from "@storybook/react-vite";

import { DatePicker } from "./date-picker";

const meta = {
  title: "design-system/Date Picker",
  tags: ["autodocs"],
  component: DatePicker,
  args: {
    label: "Starting date",
    supportText: "",
    granularity: "minute",
    className: "w-[260px]",
    isDisabled: false,
    isReadOnly: false,
    isInvalid: false,
    isRequired: false,
  },
  argTypes: {
    granularity: {
      control: { type: "select" },
      options: ["day", "minute"],
      description:
        "The granularity prop allows you to control the smallest unit that is displayed by DatePicker.",
      table: {
        type: { summary: "Granularity" },
        defaultValue: { summary: "day" },
      },
    },
  },
} satisfies Meta<typeof DatePicker>;

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
    supportText: "This field is required.",
  },
};

export const ReadOnly: Story = {
  args: {
    isReadOnly: true,
    value: parseDateTime("2022-11-07T00:45"),
  },
};

export const Clearable: Story = {
  args: {
    isClearable: true,
    defaultValue: parseDateTime("2022-11-07T00:45"),
    supportText: "Leave empty to leave this end open.",
  },
};
