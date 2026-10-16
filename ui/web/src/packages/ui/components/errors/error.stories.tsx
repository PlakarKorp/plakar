import { Meta, StoryObj } from "@storybook/react-vite";

import { Errors } from "./errors";

const meta: Meta<typeof Errors> = {
  title: "design-system/Errors",
  component: Errors,
  tags: ["autodocs"],
  parameters: {
    layout: "centered",
  },
};

export default meta;
type Story = StoryObj<typeof Errors>;

export const Default: Story = {
  args: {
    length: 3,
  },
};

export const OneError: Story = {
  args: {
    length: 1,
  },
};

export const NoErrors: Story = {
  args: {
    length: 0,
  },
  parameters: {
    docs: {
      description: {
        story:
          "This story renders nothing because `length` is <= 0 or undefined.",
      },
    },
  },
};
