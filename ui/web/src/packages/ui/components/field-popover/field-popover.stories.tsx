import { Meta, StoryObj } from "@storybook/react-vite";

import { FieldPopover } from "./field-popover";
import { NumberInput } from "../number-input/number-input";

const meta = {
  title: "design-system/Field Popover",
  tags: ["autodocs"],
  component: FieldPopover,
  args: {
    label: "Starts",
    value: "Day 1, 02:00",
    placeholder: "Choose a start",
    supportText: "",
    className: "w-48",
    isDisabled: false,
    isReadOnly: false,
    isInvalid: false,
    isRequired: false,
    children: (
      <div className="flex flex-col gap-3">
        <NumberInput label="Day" defaultValue={1} inputWidth="w-32" />
        <NumberInput label="Hour" defaultValue={2} inputWidth="w-32" />
      </div>
    ),
  },
} satisfies Meta<typeof FieldPopover>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Empty: Story = {
  args: {
    value: "",
  },
};

export const WithTitleAndFooter: Story = {
  args: {
    title: "Monthly start",
    footer: "Monthly on day 1 at 02:00, Europe/Paris",
  },
};

export const Invalid: Story = {
  args: {
    isInvalid: true,
    supportText: "This date does not exist in the chosen month.",
  },
};

export const ReadOnly: Story = {
  args: {
    isReadOnly: true,
  },
};
