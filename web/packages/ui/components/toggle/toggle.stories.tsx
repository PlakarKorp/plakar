import { Meta, StoryObj } from "@storybook/react-vite";

import { Toggle } from "./toggle";

const meta = {
  title: "design-system/Toggle",
  tags: ["autodocs"],
  component: Toggle,
} satisfies Meta<typeof Toggle>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Primary: Story = {
  args: {
    title: "Lorem ipsum",
    description: "Lorem ipsum",
    isDisabled: false,
  },
};

export const ReadOnly: Story = {
  args: {
    title: "Read-only toggle",
    isReadOnly: true,
  },
};

// A toggle whose row already names what it controls, so the label would repeat
// it. What sits beside it carries the meaning; the toggle carries the state.
export const WithoutLabel: Story = {
  args: {
    "aria-label": "Prevent deletion",
  },
};

export const Disabled: Story = {
  args: {
    title: "Cannot be changed here",
    isDisabled: true,
  },
};
